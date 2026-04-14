from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
import os
import io
import json
import base64
import logging
import requests
from dotenv import load_dotenv, find_dotenv
from PIL import Image
from google.oauth2 import service_account
from google.auth.transport.requests import Request as GoogleAuthRequest

load_dotenv(find_dotenv())

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("detail-extracter-agent")

app = FastAPI()

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

GOOGLE_SERVICE_ACCOUNT_JSON = os.environ.get("GOOGLE_SERVICE_ACCOUNT_JSON")
VISION_API_URL = "https://vision.googleapis.com/v1/images:annotate"

# Cached credentials — reused across requests, refreshed only when expired
_credentials = None


def get_vision_credentials() -> service_account.Credentials:
    """Return a valid (possibly cached) service account credential with the Vision scope."""
    global _credentials
    if _credentials is None:
        info = json.loads(GOOGLE_SERVICE_ACCOUNT_JSON)
        _credentials = service_account.Credentials.from_service_account_info(
            info,
            scopes=["https://www.googleapis.com/auth/cloud-vision"],
        )
    if not _credentials.valid:
        _credentials.refresh(GoogleAuthRequest())
    return _credentials


# Image processing

MAX_IMAGE_DIMENSION = 1024  # px — keeps base64 well under Vision API limits


def resize_image_base64(data_uri: str) -> str:
    """Resize a base64 data-URI image so its longest side is ≤ MAX_IMAGE_DIMENSION.
    Returns a data URI with the (possibly resized) JPEG image."""

    if "," in data_uri:
        header, b64data = data_uri.split(",", 1)
    else:
        header, b64data = "data:image/jpeg;base64", data_uri

    raw_bytes = base64.b64decode(b64data)
    img = Image.open(io.BytesIO(raw_bytes))

    if img.mode in ("RGBA", "P"):
        img = img.convert("RGB")

    w, h = img.size
    if max(w, h) > MAX_IMAGE_DIMENSION:
        scale = MAX_IMAGE_DIMENSION / max(w, h)
        new_size = (int(w * scale), int(h * scale))
        img = img.resize(new_size, Image.LANCZOS)
        logger.info(f"Resized image from {w}x{h} to {new_size[0]}x{new_size[1]}")

    buf = io.BytesIO()
    img.save(buf, format="JPEG", quality=85)
    resized_b64 = base64.b64encode(buf.getvalue()).decode()
    logger.info(f"Image base64 size: {len(resized_b64)} chars")
    return f"data:image/jpeg;base64,{resized_b64}"


# Cloud Vision API

def call_vision_api(image_base64: str) -> dict:
    """Call Google Cloud Vision API using service account bearer token auth."""
    if "," in image_base64:
        _, b64data = image_base64.split(",", 1)
    else:
        b64data = image_base64

    credentials = get_vision_credentials()

    payload = {
        "requests": [{
            "image": {"content": b64data},
            "features": [
                {"type": "LABEL_DETECTION", "maxResults": 15},
                {"type": "LOGO_DETECTION", "maxResults": 5},
                {"type": "TEXT_DETECTION", "maxResults": 1},
                {"type": "IMAGE_PROPERTIES"},
                {"type": "OBJECT_LOCALIZATION", "maxResults": 10},
            ],
        }]
    }

    response = requests.post(
        VISION_API_URL,
        headers={"Authorization": f"Bearer {credentials.token}"},
        json=payload,
        timeout=30,
    )
    if not response.ok:
        logger.error(f"Vision API {response.status_code}: {response.text}")
    response.raise_for_status()
    return response.json()["responses"][0]


# Categorization

# Each entry: (keyword list, (category, item_type)) — checked in order, first match wins.
LABEL_CATEGORY_MAP = [
    (
        ["backpack", "bag", "luggage", "suitcase", "handbag", "purse",
         "tote", "briefcase", "wallet", "pouch", "duffel"],
        ("Bags & Luggage", "bag"),
    ),
    (
        ["phone", "smartphone", "mobile", "tablet", "laptop", "computer",
         "headphone", "earphone", "earbuds", "camera", "charger", "cable",
         "electronic", "device", "gadget", "keyboard", "mouse", "remote",
         "speaker", "battery"],
        ("Electronics", "electronics"),
    ),
    (
        ["jacket", "coat", "shirt", "pants", "trousers", "clothing", "shoe",
         "boot", "hat", "cap", "scarf", "glove", "umbrella", "watch",
         "glasses", "sunglasses", "jewelry", "ring", "necklace", "bracelet",
         "belt", "tie", "sock", "sweater", "hoodie"],
        ("Clothing & Accessories", "clothing"),
    ),
    (
        ["passport", "id card", "card", "document", "paper", "license",
         "certificate", "folder", "envelope"],
        ("Documents & Cards", "document"),
    ),
    (
        ["key", "keychain", "keyring"],
        ("Keys", "keys"),
    ),
    (
        ["bottle", "container", "flask", "cup", "mug", "thermos",
         "tumbler", "canteen"],
        ("Bottles & Containers", "bottle"),
    ),
    (
        ["book", "notebook", "magazine", "newspaper", "stationery",
         "pen", "pencil", "binder"],
        ("Books & Stationery", "book"),
    ),
    (
        ["toy", "game", "stuffed animal", "doll", "action figure", "ball",
         "plush", "puzzle"],
        ("Toys & Games", "toy"),
    ),
]


def categorize_from_labels(labels: list) -> tuple:
    """Map Cloud Vision label strings to a (category, item_type) pair."""
    labels_lower = [l.lower() for l in labels]
    for keywords, result in LABEL_CATEGORY_MAP:
        for kw in keywords:
            if any(kw in label for label in labels_lower):
                return result
    return "Other", "other"


# Color extraction

NAMED_COLORS = [
    ("black",   (0,   0,   0)),
    ("white",   (255, 255, 255)),
    ("gray",    (128, 128, 128)),
    ("red",     (220,  20,  60)),
    ("orange",  (255, 140,   0)),
    ("yellow",  (255, 215,   0)),
    ("green",   ( 34, 139,  34)),
    ("blue",    ( 30, 144, 255)),
    ("navy",    (  0,   0, 128)),
    ("purple",  (128,   0, 128)),
    ("pink",    (255, 105, 180)),
    ("brown",   (139,  69,  19)),
    ("beige",   (245, 245, 220)),
    ("silver",  (192, 192, 192)),
    ("gold",    (218, 165,  32)),
]


def rgb_to_name(r: int, g: int, b: int) -> str:
    """Return the nearest named color for an RGB triplet."""
    best_name, best_dist = "unknown", float("inf")
    for name, (nr, ng, nb) in NAMED_COLORS:
        dist = (r - nr) ** 2 + (g - ng) ** 2 + (b - nb) ** 2
        if dist < best_dist:
            best_dist = dist
            best_name = name
    return best_name


def extract_dominant_color(vision_response: dict) -> str:
    """Return the top-scored dominant color name from imagePropertiesAnnotation."""
    colors = (
        vision_response
        .get("imagePropertiesAnnotation", {})
        .get("dominantColors", {})
        .get("colors", [])
    )
    if not colors:
        return "unknown"
    top = max(colors, key=lambda c: c.get("score", 0))
    rgb = top.get("color", {})
    return rgb_to_name(
        int(rgb.get("red", 0)),
        int(rgb.get("green", 0)),
        int(rgb.get("blue", 0)),
    )


# Detail extraction

MATERIAL_KEYWORDS = {
    "leather": "leather",
    "metal":   "metal",
    "plastic": "plastic",
    "fabric":  "fabric",
    "nylon":   "nylon",
    "canvas":  "canvas",
    "glass":   "glass",
    "wood":    "wood",
    "rubber":  "rubber",
    "paper":   "paper",
    "cotton":  "cotton",
}


def extract_details(vision_response: dict) -> dict:
    """Derive structured item details from a Cloud Vision API response."""

    # Labels (confidence >= 0.6)
    label_annotations = vision_response.get("labelAnnotations", [])
    labels = [a["description"] for a in label_annotations if a.get("score", 0) >= 0.6]

    # Localized objects (confidence >= 0.5) — supplement labels
    objects = [
        o["name"] for o in vision_response.get("localizedObjectAnnotations", [])
        if o.get("score", 0) >= 0.5
    ]

    # Deduplicate while preserving confidence order
    all_labels = list(dict.fromkeys(labels + objects))

    # Brand from logo detection
    logo_annotations = vision_response.get("logoAnnotations", [])
    brand = logo_annotations[0]["description"] if logo_annotations else "unknown"

    # OCR text — only include if it looks like meaningful text, not fragmented noise
    text_annotations = vision_response.get("textAnnotations", [])
    detected_text = ""
    if text_annotations:
        raw_text = text_annotations[0]["description"].strip()
        tokens = raw_text.split()
        if tokens:
            real_words = [t for t in tokens if len(t) >= 3]
            # Require at least 3 real words and >40% of tokens to be real words
            if len(real_words) >= 3 and len(real_words) / len(tokens) > 0.4:
                detected_text = raw_text
                if len(detected_text) > 200:
                    detected_text = detected_text[:200] + "…"

    # Dominant color
    color = extract_dominant_color(vision_response)

    # Category and type
    category, item_type = categorize_from_labels(all_labels)

    # item_name: "Brand TopLabel" or just "TopLabel"
    top_label = all_labels[0] if all_labels else "Item"
    item_name = f"{brand} {top_label}" if brand != "unknown" else top_label

    # Material inferred from labels
    material = "unknown"
    for label in all_labels:
        for kw, mat in MATERIAL_KEYWORDS.items():
            if kw in label.lower():
                material = mat
                break
        if material != "unknown":
            break

    # Build item_description with all non-explicit fields (type, brand, color, material,
    # visual features, OCR text) so staff only need to fill in name, location, route, date.
    desc_parts = []

    opening = " ".join(t for t in [color, material, item_type] if t and t != "unknown")
    if opening:
        desc_parts.append(opening.capitalize() + ".")

    if brand != "unknown":
        desc_parts.append(f"Brand: {brand}.")

    skip = {top_label.lower(), item_type.lower()}
    feature_labels = [l for l in all_labels[1:8] if l.lower() not in skip]
    if feature_labels:
        desc_parts.append(f"Visual features: {', '.join(feature_labels)}.")

    if detected_text:
        desc_parts.append(f"Text visible on item: {detected_text}.")

    item_description = " ".join(desc_parts) if desc_parts else f"{item_type.capitalize()} item."

    return {
        "item_name": item_name,
        "item_type": item_type,
        "category": category,
        "brand": brand,
        "model": "unknown",
        "color": color,
        "material": material,
        "item_condition": "unknown",
        "item_description": item_description,
    }


def run_extraction(image_base64: str) -> dict:
    """Full pipeline: resize → Cloud Vision API → extract structured details."""
    if not image_base64.startswith("data:"):
        image_base64 = f"data:image/jpeg;base64,{image_base64}"
    image_base64 = resize_image_base64(image_base64)

    vision_response = call_vision_api(image_base64)
    logger.info(f"Vision API response keys: {list(vision_response.keys())}")

    return extract_details(vision_response)


# API endpoints

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
            "logos":  [a["description"] for a in vision_response.get("logoAnnotations", [])],
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
