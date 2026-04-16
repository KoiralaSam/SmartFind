from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
import requests

from extractor import (
    GOOGLE_SERVICE_ACCOUNT_JSON,
    call_vision_api,
    extract_details,
    logger,
    resize_image_base64,
    run_extraction,
)
from grpc_handler import start_grpc_server, stop_grpc_server

app = FastAPI()

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.on_event("startup")
def _startup() -> None:
    start_grpc_server()


@app.on_event("shutdown")
def _shutdown() -> None:
    stop_grpc_server()


class ExtractRequest(BaseModel):
    image_base64: str


class ExtractResponse(BaseModel):
    item_name: str = "unknown"
    item_type: str = "unknown"
    category: str = "Other"
    brand: str = "unknown"
    model: str = "unknown"
    color: str = "unknown"
    material: str = "unknown"
    item_condition: str = "unknown"
    item_description: str = ""


@app.get("/health")
def health():
    return "Detail extracter agent is running!"


@app.post("/test-extract")
def test_extract(req: ExtractRequest):
    """Diagnostic endpoint: tests each step of the extraction pipeline individually
    and reports which steps pass or fail."""
    results = {
        "service_account_configured": bool(GOOGLE_SERVICE_ACCOUNT_JSON),
        "image_received": bool(req.image_base64),
        "image_size_bytes": len(req.image_base64),
        "steps": {},
    }

    if not results["service_account_configured"]:
        results["overall"] = "fail"
        results["error"] = "GOOGLE_SERVICE_ACCOUNT_JSON is not configured"
        return results

    image = req.image_base64
    if not image.startswith("data:"):
        image = f"data:image/jpeg;base64,{image}"

    try:
        image = resize_image_base64(image)
        results["resized_image_size"] = len(image)
    except Exception as e:
        results["overall"] = "fail"
        results["error"] = f"Image processing failed: {e}"
        return results

    # Step 1: Cloud Vision API
    try:
        vision_response = call_vision_api(image)
        results["steps"]["vision_api"] = {
            "status": "ok",
            "labels": [a["description"] for a in vision_response.get("labelAnnotations", [])[:5]],
            "logos": [a["description"] for a in vision_response.get("logoAnnotations", [])],
        }
    except Exception as e:
        results["steps"]["vision_api"] = {"status": "fail", "error": str(e)}
        results["overall"] = "fail"
        return results

    # Step 2: Detail extraction
    try:
        structured = extract_details(vision_response)
        results["steps"]["extract_details"] = {"status": "ok", "output": structured}
    except Exception as e:
        results["steps"]["extract_details"] = {"status": "fail", "error": str(e)}
        results["overall"] = "fail"
        return results

    results["overall"] = "ok"
    return results


@app.post("/extract", response_model=ExtractResponse)
def extract(req: ExtractRequest):
    if not GOOGLE_SERVICE_ACCOUNT_JSON:
        raise HTTPException(status_code=500, detail="GOOGLE_SERVICE_ACCOUNT_JSON not configured")
    try:
        result = run_extraction(req.image_base64)
        return ExtractResponse(**result)
    except requests.HTTPError as e:
        logger.error(f"Cloud Vision API error: {e}")
        raise HTTPException(status_code=502, detail=f"Cloud Vision API error: {e}")
    except Exception as e:
        logger.error(f"Extraction error: {e}")
        raise HTTPException(status_code=500, detail=str(e))

