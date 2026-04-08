from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
import os
import io
import base64
import json
import logging
from dotenv import load_dotenv, find_dotenv
from groq import Groq
from PIL import Image

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

client = Groq(api_key=os.environ.get("GROQ_API_KEY"))

# ────────────────────────────────────────────────────────────────
#  IMAGE PROCESSING
# ────────────────────────────────────────────────────────────────

MAX_IMAGE_DIMENSION = 1024  # px — keeps base64 well under Groq limits


def resize_image_base64(data_uri: str) -> str:
    """Resize a base64 data-URI image so its longest side is ≤ MAX_IMAGE_DIMENSION.
    Returns a data URI with the (possibly resized) JPEG image."""

    # Strip the data URI header to get raw base64
    if "," in data_uri:
        header, b64data = data_uri.split(",", 1)
    else:
        header, b64data = "data:image/jpeg;base64", data_uri

    raw_bytes = base64.b64decode(b64data)
    img = Image.open(io.BytesIO(raw_bytes))

    # Convert RGBA/P to RGB for JPEG
    if img.mode in ("RGBA", "P"):
        img = img.convert("RGB")

    # Resize if needed
    w, h = img.size
    if max(w, h) > MAX_IMAGE_DIMENSION:
        scale = MAX_IMAGE_DIMENSION / max(w, h)
        new_size = (int(w * scale), int(h * scale))
        img = img.resize(new_size, Image.LANCZOS)
        logger.info(f"Resized image from {w}x{h} to {new_size[0]}x{new_size[1]}")

    # Re-encode as JPEG
    buf = io.BytesIO()
    img.save(buf, format="JPEG", quality=85)
    resized_b64 = base64.b64encode(buf.getvalue()).decode()
    logger.info(f"Image base64 size: {len(resized_b64)} chars")
    return f"data:image/jpeg;base64,{resized_b64}"


# ────────────────────────────────────────────────────────────────
#  TOOLS — each tool is a function the agent can invoke
# ────────────────────────────────────────────────────────────────

TOOL_DEFINITIONS = [
    {
        "type": "function",
        "function": {
            "name": "analyze_image",
            "description": (
                "Perform initial visual analysis of the uploaded item image. "
                "Returns a free-form text description of everything visible."
            ),
            "parameters": {
                "type": "object",
                "properties": {
                    "focus": {
                        "type": "string",
                        "description": "What to focus on: 'general' for overall look, 'branding' for logos/text, 'condition' for wear/damage",
                    }
                },
                "required": ["focus"],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "categorize_item",
            "description": (
                "Based on the image analysis, determine the item category and type. "
                "Choose from standard transit lost & found categories."
            ),
            "parameters": {
                "type": "object",
                "properties": {
                    "visual_description": {
                        "type": "string",
                        "description": "The visual description from the image analysis step",
                    }
                },
                "required": ["visual_description"],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "extract_structured_details",
            "description": (
                "Compile all gathered information into a final structured JSON object "
                "with item_name, item_type, category, brand, model, color, material, "
                "item_condition, and item_description."
            ),
            "parameters": {
                "type": "object",
                "properties": {
                    "visual_description": {
                        "type": "string",
                        "description": "Full visual description from analysis",
                    },
                    "category": {
                        "type": "string",
                        "description": "Category determined by categorize_item",
                    },
                    "item_type": {
                        "type": "string",
                        "description": "Item type determined by categorize_item",
                    },
                },
                "required": ["visual_description", "category", "item_type"],
            },
        },
    },
]


def tool_analyze_image(image_base64: str, focus: str) -> str:
    """Vision model call — looks at the image and returns a text description."""
    focus_prompts = {
        "general": "Describe everything you see: the item type, color, size, shape, any visible features.",
        "branding": "Focus on any brand names, logos, text, labels, model numbers, or serial numbers visible on the item.",
        "condition": "Describe the condition of the item: is it new, worn, damaged? Any scratches, stains, tears, or missing parts?",
    }
    prompt = focus_prompts.get(focus, focus_prompts["general"])

    messages = [
        {
            "role": "user",
            "content": [
                {"type": "text", "text": prompt},
                {"type": "image_url", "image_url": {"url": image_base64}},
            ],
        },
    ]

    try:
        completion = client.chat.completions.create(
            model="meta-llama/llama-4-scout-17b-16e-instruct",
            messages=messages,
            temperature=0.2,
            max_tokens=512,
        )
        return completion.choices[0].message.content.strip()
    except Exception as e:
        logger.error(f"Groq vision API error: {e}")
        raise


CATEGORIES = [
    "Bags & Luggage", "Electronics", "Clothing & Accessories",
    "Documents & Cards", "Keys", "Bottles & Containers",
    "Books & Stationery", "Toys & Games", "Other",
]

ITEM_TYPES = [
    "bag", "electronics", "clothing", "accessory", "document",
    "keys", "bottle", "book", "toy", "other",
]


def tool_categorize_item(visual_description: str) -> dict:
    """Text model call — categorizes the item from its description."""
    messages = [
        {
            "role": "system",
            "content": (
                "You categorize lost & found items for a transit system.\n"
                f"Valid categories: {', '.join(CATEGORIES)}\n"
                f"Valid item types: {', '.join(ITEM_TYPES)}\n\n"
                "Return ONLY a JSON object: {\"category\": \"...\", \"item_type\": \"...\"}"
            ),
        },
        {
            "role": "user",
            "content": f"Categorize this item based on the description:\n\n{visual_description}",
        },
    ]

    completion = client.chat.completions.create(
        model="llama-3.3-70b-versatile",
        messages=messages,
        temperature=0.1,
        max_tokens=100,
    )
    reply = completion.choices[0].message.content.strip()
    return _parse_json(reply)


def tool_extract_structured_details(
    visual_description: str, category: str, item_type: str
) -> dict:
    """Text model call — compiles everything into the final structured output."""
    messages = [
        {
            "role": "system",
            "content": (
                "You are a detail extraction agent for a transit lost & found system.\n"
                "Given a visual description, category, and item type, produce a structured JSON.\n\n"
                "Return ONLY a JSON object with these fields (use \"unknown\" if not identifiable):\n"
                "{\n"
                "  \"item_name\": \"short descriptive name, e.g. 'Black Nike Backpack'\",\n"
                "  \"item_type\": \"<provided item_type>\",\n"
                "  \"category\": \"<provided category>\",\n"
                "  \"brand\": \"brand if identified\",\n"
                "  \"model\": \"model if identified\",\n"
                "  \"color\": \"primary color(s)\",\n"
                "  \"material\": \"material type, e.g. leather, nylon, plastic, metal\",\n"
                "  \"item_condition\": \"good, worn, damaged, or new\",\n"
                "  \"item_description\": \"detailed description with distinguishing features\"\n"
                "}"
            ),
        },
        {
            "role": "user",
            "content": (
                f"Category: {category}\n"
                f"Item type: {item_type}\n\n"
                f"Visual description:\n{visual_description}"
            ),
        },
    ]

    completion = client.chat.completions.create(
        model="llama-3.3-70b-versatile",
        messages=messages,
        temperature=0.1,
        max_tokens=512,
    )
    reply = completion.choices[0].message.content.strip()
    return _parse_json(reply)


# ────────────────────────────────────────────────────────────────
#  AGENT — orchestrates multi-step tool-use reasoning loop
# ────────────────────────────────────────────────────────────────

AGENT_SYSTEM_PROMPT = """You are a detail extraction agent for a transit lost & found system.
Your job is to analyze an image of a found item and extract structured details.

You have access to these tools:
1. analyze_image  — visually inspect the image (use focus: "general", "branding", or "condition")
2. categorize_item — determine the item's category and type from a description
3. extract_structured_details — compile all findings into a final structured JSON

STRATEGY:
- Step 1: Call analyze_image with focus "general" to get an overall description
- Step 2: Call analyze_image with focus "branding" to look for brand/model info
- Step 3: Call categorize_item with the combined description
- Step 4: Call extract_structured_details with all gathered information

Always follow all steps. Call one tool at a time. After extract_structured_details, you are done."""

MAX_AGENT_STEPS = 6


def run_agent(image_base64: str) -> dict:
    """Run the agent loop: the LLM decides which tools to call and when."""

    # Ensure proper data URI format and resize for the vision tool
    if not image_base64.startswith("data:"):
        image_base64 = f"data:image/jpeg;base64,{image_base64}"
    image_base64 = resize_image_base64(image_base64)

    messages = [
        {"role": "system", "content": AGENT_SYSTEM_PROMPT},
        {
            "role": "user",
            "content": "Analyze the uploaded item image and extract all details. Begin with step 1.",
        },
    ]

    final_result = None

    for step in range(MAX_AGENT_STEPS):
        logger.info(f"Agent step {step + 1}/{MAX_AGENT_STEPS}")

        completion = client.chat.completions.create(
            model="llama-3.3-70b-versatile",
            messages=messages,
            tools=TOOL_DEFINITIONS,
            tool_choice="auto",
            temperature=0.1,
            max_tokens=512,
        )

        response_message = completion.choices[0].message
        messages.append(response_message)

        # If no tool calls, the agent is done reasoning
        if not response_message.tool_calls:
            logger.info("Agent finished — no more tool calls")
            break

        # Execute each tool call
        for tool_call in response_message.tool_calls:
            fn_name = tool_call.function.name
            fn_args = json.loads(tool_call.function.arguments)

            logger.info(f"Tool call: {fn_name}({json.dumps(fn_args)[:100]}...)")

            if fn_name == "analyze_image":
                result = tool_analyze_image(image_base64, fn_args.get("focus", "general"))
                tool_result = result

            elif fn_name == "categorize_item":
                result = tool_categorize_item(fn_args["visual_description"])
                tool_result = json.dumps(result)

            elif fn_name == "extract_structured_details":
                result = tool_extract_structured_details(
                    fn_args["visual_description"],
                    fn_args.get("category", "Other"),
                    fn_args.get("item_type", "other"),
                )
                final_result = result
                tool_result = json.dumps(result)

            else:
                tool_result = f"Unknown tool: {fn_name}"

            messages.append({
                "role": "tool",
                "tool_call_id": tool_call.id,
                "content": tool_result,
            })

        # If we got a final result from extract_structured_details, we're done
        if final_result is not None:
            logger.info("Agent completed — structured details extracted")
            break

    # Fallback: if agent never called extract_structured_details, do a simple extraction
    if final_result is None:
        logger.warning("Agent did not produce structured output — running fallback")
        final_result = fallback_extract(image_base64)

    return final_result


def fallback_extract(image_base64: str) -> dict:
    """Single-shot fallback if the agent loop doesn't produce a result."""
    image_base64 = resize_image_base64(image_base64)
    description = tool_analyze_image(image_base64, "general")
    categorization = tool_categorize_item(description)
    return tool_extract_structured_details(
        description,
        categorization.get("category", "Other"),
        categorization.get("item_type", "other"),
    )


# ────────────────────────────────────────────────────────────────
#  HELPERS
# ────────────────────────────────────────────────────────────────

def _parse_json(text: str) -> dict:
    """Parse JSON from LLM response, handling markdown code blocks."""
    text = text.strip()
    if text.startswith("```"):
        lines = text.split("\n")
        text = "\n".join(lines[1:-1])
    return json.loads(text)


# ────────────────────────────────────────────────────────────────
#  API ENDPOINTS
# ────────────────────────────────────────────────────────────────

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
        "groq_api_key_set": bool(os.environ.get("GROQ_API_KEY")),
        "image_received": bool(req.image_base64),
        "image_size_bytes": len(req.image_base64),
        "steps": {},
    }

    if not results["groq_api_key_set"]:
        results["overall"] = "fail"
        results["error"] = "GROQ_API_KEY is not configured"
        return results

    image = req.image_base64
    if not image.startswith("data:"):
        image = f"data:image/jpeg;base64,{image}"

    # Resize image to stay within Groq API limits
    try:
        image = resize_image_base64(image)
        results["resized_image_size"] = len(image)
    except Exception as e:
        results["overall"] = "fail"
        results["error"] = f"Image processing failed: {e}"
        return results

    # Step 1: Vision analysis
    try:
        description = tool_analyze_image(image, "general")
        results["steps"]["analyze_image"] = {"status": "ok", "output": description[:300]}
    except Exception as e:
        results["steps"]["analyze_image"] = {"status": "fail", "error": str(e)}
        results["overall"] = "fail"
        return results

    # Step 2: Categorization
    try:
        categorization = tool_categorize_item(description)
        results["steps"]["categorize_item"] = {"status": "ok", "output": categorization}
    except Exception as e:
        results["steps"]["categorize_item"] = {"status": "fail", "error": str(e)}
        results["overall"] = "fail"
        return results

    # Step 3: Structured extraction
    try:
        structured = tool_extract_structured_details(
            description,
            categorization.get("category", "Other"),
            categorization.get("item_type", "other"),
        )
        results["steps"]["extract_structured_details"] = {"status": "ok", "output": structured}
    except Exception as e:
        results["steps"]["extract_structured_details"] = {"status": "fail", "error": str(e)}
        results["overall"] = "fail"
        return results

    results["overall"] = "ok"
    return results


@app.post("/extract", response_model=ExtractResponse)
def extract(req: ExtractRequest):
    if not os.environ.get("GROQ_API_KEY"):
        raise HTTPException(status_code=500, detail="GROQ_API_KEY not configured")
    try:
        result = run_agent(req.image_base64)
        return ExtractResponse(**result)
    except json.JSONDecodeError:
        raise HTTPException(status_code=500, detail="Failed to parse AI response as JSON")
    except Exception as e:
        logger.error(f"Agent error: {e}")
        raise HTTPException(status_code=500, detail=str(e))
