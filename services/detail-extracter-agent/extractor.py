import base64
import io
import json
import logging
import os
from typing import Any

import requests
from dotenv import find_dotenv, load_dotenv
from google.auth.transport.requests import Request as GoogleAuthRequest
from google.oauth2 import service_account
from PIL import Image

load_dotenv(find_dotenv())

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("detail-extracter-agent")

GOOGLE_SERVICE_ACCOUNT_JSON = os.environ.get("GOOGLE_SERVICE_ACCOUNT_JSON")
VISION_API_URL = "https://vision.googleapis.com/v1/images:annotate"
OPENAI_API_KEY = os.environ.get("OPENAI_API_KEY", "").strip()
OPENAI_EXTRACT_MODEL = (os.environ.get("OPENAI_EXTRACT_MODEL") or "gpt-4o-mini").strip()

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
        _, b64data = data_uri.split(",", 1)
    else:
        b64data = data_uri

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
        "requests": [
            {
                "image": {"content": b64data},
                "features": [
                    {"type": "LABEL_DETECTION", "maxResults": 15},
                    {"type": "LOGO_DETECTION", "maxResults": 5},
                    {"type": "TEXT_DETECTION", "maxResults": 1},
                    {"type": "IMAGE_PROPERTIES"},
                    {"type": "OBJECT_LOCALIZATION", "maxResults": 10},
                ],
            }
        ]
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
        [
            "backpack",
            "bag",
            "luggage",
            "suitcase",
            "handbag",
            "purse",
            "tote",
            "briefcase",
            "wallet",
            "pouch",
            "duffel",
        ],
        ("Bags & Luggage", "bag"),
    ),
    (
        [
            "phone",
            "smartphone",
            "mobile",
            "tablet",
            "laptop",
            "computer",
            "headphone",
            "earphone",
            "earbuds",
            "camera",
            "charger",
            "cable",
            "electronic",
            "device",
            "gadget",
            "keyboard",
            "mouse",
            "remote",
            "speaker",
            "battery",
        ],
        ("Electronics", "electronics"),
    ),
    (
        [
            "jacket",
            "coat",
            "shirt",
            "pants",
            "trousers",
            "clothing",
            "shoe",
            "boot",
            "hat",
            "cap",
            "scarf",
            "glove",
            "umbrella",
            "watch",
            "glasses",
            "sunglasses",
            "jewelry",
            "ring",
            "necklace",
            "bracelet",
            "belt",
            "tie",
            "sock",
            "sweater",
            "hoodie",
        ],
        ("Clothing & Accessories", "clothing"),
    ),
    (
        ["passport", "id card", "card", "document", "paper", "license", "certificate", "folder", "envelope"],
        ("Documents & Cards", "document"),
    ),
    (["key", "keychain", "keyring"], ("Keys", "keys")),
    (
        ["bottle", "container", "flask", "cup", "mug", "thermos", "tumbler", "canteen"],
        ("Bottles & Containers", "bottle"),
    ),
    (
        ["book", "notebook", "magazine", "newspaper", "stationery", "pen", "pencil", "binder"],
        ("Books & Stationery", "book"),
    ),
    (
        ["toy", "game", "stuffed animal", "doll", "action figure", "ball", "plush", "puzzle"],
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
    ("black", (0, 0, 0)),
    ("white", (255, 255, 255)),
    ("gray", (128, 128, 128)),
    ("red", (220, 20, 60)),
    ("orange", (255, 140, 0)),
    ("yellow", (255, 215, 0)),
    ("green", (34, 139, 34)),
    ("blue", (30, 144, 255)),
    ("navy", (0, 0, 128)),
    ("purple", (128, 0, 128)),
    ("pink", (255, 105, 180)),
    ("brown", (139, 69, 19)),
    ("beige", (245, 245, 220)),
    ("silver", (192, 192, 192)),
    ("gold", (218, 165, 32)),
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
        vision_response.get("imagePropertiesAnnotation", {})
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
    "metal": "metal",
    "plastic": "plastic",
    "fabric": "fabric",
    "nylon": "nylon",
    "canvas": "canvas",
    "glass": "glass",
    "wood": "wood",
    "rubber": "rubber",
    "paper": "paper",
    "cotton": "cotton",
}

EXTRACTION_FIELDS = (
    "item_name",
    "category",
    "color",
    "brand",
    "model",
    "material",
    "item_condition",
    "item_description",
)


def extract_details(vision_response: dict) -> dict:
    """Derive structured item details from a Cloud Vision API response."""

    # Labels (confidence >= 0.6)
    label_annotations = vision_response.get("labelAnnotations", [])
    labels = [a["description"] for a in label_annotations if a.get("score", 0) >= 0.6]

    # Localized objects (confidence >= 0.5) — supplement labels
    objects = [
        o["name"]
        for o in vision_response.get("localizedObjectAnnotations", [])
        if o.get("score", 0) >= 0.5
    ]

    # Deduplicate while preserving confidence order
    all_labels = list(dict.fromkeys(labels + objects))

    # Brand from logo detection
    logo_annotations = vision_response.get("logoAnnotations", [])
    brand = logo_annotations[0]["description"] if logo_annotations else ""

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
    if color == "unknown":
        color = ""

    # Category and type
    category, item_type = categorize_from_labels(all_labels)

    # item_name: "Brand TopLabel" or just "TopLabel"
    top_label = all_labels[0] if all_labels else "Item"
    item_name = f"{brand} {top_label}" if brand else top_label

    # Material inferred from labels
    material = ""
    for label in all_labels:
        for kw, mat in MATERIAL_KEYWORDS.items():
            if kw in label.lower():
                material = mat
                break
        if material:
            break

    # Build item_description with all non-explicit fields (type, brand, color, material,
    # visual features, OCR text) so staff only need to fill in name, location, route, date.
    desc_parts = []

    opening = " ".join(t for t in [color, material, item_type] if t)
    if opening:
        desc_parts.append(opening.capitalize() + ".")

    if brand:
        desc_parts.append(f"Brand: {brand}.")

    skip = {top_label.lower(), item_type.lower()}
    feature_labels = [l for l in all_labels[1:8] if l.lower() not in skip]
    if feature_labels:
        desc_parts.append(f"Visual features: {', '.join(feature_labels)}.")

    if detected_text:
        desc_parts.append(f"Text visible on item: {detected_text}.")

    item_description = " ".join(desc_parts) if desc_parts else f"{item_type.capitalize()} item."

    return {
        "item_name": item_name if item_name != "Item" else "",
        "item_type": item_type if item_type != "other" else "",
        "category": category if category != "Other" else "",
        "brand": brand,
        "model": "",
        "color": color,
        "material": material,
        "item_condition": "",
        "item_description": item_description,
    }


def _normalize_extraction_result(data: dict[str, Any]) -> dict[str, str]:
    normalized: dict[str, str] = {}
    for field in EXTRACTION_FIELDS:
        value = data.get(field, "")
        text = str(value or "").strip()
        if text.lower() == "unknown":
            text = ""
        normalized[field] = text
    return normalized


def _llm_extract_details(vision_response: dict, heuristic: dict[str, str]) -> dict[str, str]:
    if not OPENAI_API_KEY:
        return heuristic

    labels = [
        a["description"]
        for a in vision_response.get("labelAnnotations", [])
        if a.get("description")
    ]
    logos = [
        a["description"]
        for a in vision_response.get("logoAnnotations", [])
        if a.get("description")
    ]
    objects = [
        a["name"]
        for a in vision_response.get("localizedObjectAnnotations", [])
        if a.get("name")
    ]
    text_annotations = vision_response.get("textAnnotations", [])
    raw_text = text_annotations[0]["description"].strip() if text_annotations else ""

    system_prompt = """You normalize lost-and-found image detection into staff form fields.

Return ONLY valid JSON with exactly these string keys:
- item_name
- category
- color
- brand
- model
- material
- item_condition
- item_description

Rules:
- Resolve findings into those fields only.
- Any field you cannot reliably determine must be an empty string.
- Do not use the words "unknown", "n/a", or null.
- item_name should be a short human-friendly name like "Apple MacBook Pro" or "Black backpack".
- category should match one of:
  Bags & Luggage
  Electronics
  Clothing & Accessories
  Documents & Cards
  Keys
  Bottles & Containers
  Books & Stationery
  Toys & Games
  Other
- item_description should be concise but useful for staff.
- Do not invent a brand, model, material, or condition unless there is evidence in the vision findings.
"""

    user_prompt = {
        "heuristic_guess": heuristic,
        "vision_findings": {
            "labels": labels[:12],
            "logos": logos[:5],
            "objects": objects[:10],
            "ocr_text": raw_text[:300],
            "dominant_colors": vision_response.get("imagePropertiesAnnotation", {})
            .get("dominantColors", {})
            .get("colors", [])[:5],
        },
    }

    response = requests.post(
        "https://api.openai.com/v1/chat/completions",
        headers={
            "Authorization": f"Bearer {OPENAI_API_KEY}",
            "Content-Type": "application/json",
        },
        json={
            "model": OPENAI_EXTRACT_MODEL,
            "temperature": 0,
            "response_format": {"type": "json_object"},
            "messages": [
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": json.dumps(user_prompt)},
            ],
        },
        timeout=30,
    )
    response.raise_for_status()
    content = response.json()["choices"][0]["message"]["content"]
    parsed = json.loads(content)
    return _normalize_extraction_result(parsed)


def run_extraction(image_base64: str) -> dict:
    """Full pipeline: resize → Cloud Vision API → extract structured details."""
    if not image_base64.startswith("data:"):
        image_base64 = f"data:image/jpeg;base64,{image_base64}"
    image_base64 = resize_image_base64(image_base64)

    vision_response = call_vision_api(image_base64)
    logger.info(f"Vision API response keys: {list(vision_response.keys())}")

    heuristic = _normalize_extraction_result(extract_details(vision_response))
    try:
        return _llm_extract_details(vision_response, heuristic)
    except Exception as e:
        logger.warning(f"LLM normalization failed, using heuristic extraction: {e}")
        return heuristic

