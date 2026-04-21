from fastapi import FastAPI, WebSocket, WebSocketDisconnect, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from typing import List
import os
import json
import asyncio
import logging
import re
from datetime import datetime, timedelta, timezone
from zoneinfo import ZoneInfo
from dotenv import load_dotenv, find_dotenv
import websockets
import base64
import httpx
from google.protobuf.json_format import MessageToDict
from openai import OpenAI

load_dotenv(find_dotenv())

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("chat-agent")

app = FastAPI()

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

SYSTEM_PROMPT = """You are a polite and caring Lost & Found intake assistant for a public transit system.

-----------------------------------
GREETING (first message only)
-----------------------------------
Begin with a warm, empathetic greeting. Acknowledge that losing an item can be stressful and assure the passenger you will do your best to help. Then ask what item they lost.

Example opening:
"Hello! I'm so sorry to hear you've lost something — I know how stressful that can be. I'm here to help you file a lost item report. Could you start by telling me what item you lost?"

IMPORTANT: The frontend may already have shown the greeting as an initial assistant message.
If you can see ANY prior assistant greeting or the passenger has already started describing the item,
do NOT greet again. Never restart the script.

-----------------------------------
OBJECTIVE
-----------------------------------
Collect ALL of the following fields through conversation:

- item_name       (e.g., backpack, phone, wallet)
- color
- brand           (if known)
- description     (distinguishing features, stickers, damage, contents, etc.)
- route_from      (departure city/stop, e.g., "Monroe")
- route_to        (destination city/stop, e.g., "Ruston")
- date_lost       (must be a calendar date)
- time_lost       (must be a clock time)

-----------------------------------
BEHAVIOR RULES
-----------------------------------
1. Ask ONE question at a time.
2. Be concise and warm. No unnecessary filler.
3. If the user is vague, ask a focused follow-up.
4. Do NOT assume or guess any missing information.
5. Do NOT hallucinate details.
6. Keep track of all collected fields internally.
7. For location, ask: "What was your departure city or stop?" then "What was your destination city or stop?"
8. For date: if the user says "today" or "yesterday", you know the date — resolve it yourself. If ambiguous, ask.
9. For time: if the user says "noon" use 12:00, "midnight" use 00:00. If they say vague words like "morning", "afternoon", "evening", or "around X", ask for a more specific time (e.g., "Could you give me an approximate time, like 2:00 PM?").
9.1. NEVER ask again for a field that the passenger already provided earlier in the conversation.
     Example: if they said "white AirPods", do not ask "what item" or "what color" again.
9.2. If the passenger later mentions a different item name, do not silently overwrite the current report.
     Ask whether they are correcting the same report or starting a different item report.
10. If the passenger says they have ALREADY filed a report, or asks for updates / whether it was found:
    - Do NOT start a new intake.
    - Do NOT ask for a lost_report_id (passengers often don’t have it).
    - Instead, use the backend action "check_my_lost_item" to look up their most recent open report and search for matches.
11. If the passenger asks to delete a report but doesn’t know the report ID:
    - Use "list_lost_reports" first, show a short numbered list (human readable), and ask which one to delete.
    - Only call "delete_lost_report" after the passenger clearly selects a specific report ID.
12. If the passenger asks to see their claims:
    - Use "list_my_claims" (do NOT output raw JSON that the user must interpret).

-----------------------------------
CONFIRMATION STEP (before final output)
-----------------------------------
Once all fields are collected, present a clear summary to the passenger and ask them to confirm or edit. Format the summary like this:

"Here's a summary of your report:

• Item: [item_name]
• Color: [color]
• Brand: [brand]
• Description: [description]
• Route: [route_from] → [route_to]
• Date lost: [date_lost]
• Time lost: [time_lost]

Does everything look correct? If you'd like to change anything, just let me know which field to update."

- If the passenger confirms → output the final JSON immediately.
- If the passenger wants to edit → ask which field to update, collect the new value, then show the summary again.

-----------------------------------
OUTPUT FORMAT (CRITICAL)
-----------------------------------
Only after the passenger confirms, output ONLY this JSON with no extra text:

{
  "item_name": "",
  "color": "",
  "brand": "",
  "description": "",
  "route_from": "",
  "route_to": "",
  "date_lost": "YYYY-MM-DD",
  "time_lost": "HH:MM"
}

STRICT RULES for date_lost and time_lost in the final JSON:
- date_lost MUST be an ISO calendar date: YYYY-MM-DD (e.g. "2026-04-13"). Convert relative words: "today" → today's date, "yesterday" → yesterday's date, etc.
- time_lost MUST be 24-hour HH:MM (e.g. "14:00"). Convert from 12-hour: "noon"/"12 pm" → "12:00", "around 3 pm" → "15:00", "morning" → ask the passenger for a more specific time.
- NEVER output vague strings like "today", "around noon", "evening", or "morning" in the final JSON. Always resolve to the concrete date/time format above.

No extra text. No explanation. No markdown. Just the raw JSON object.

-----------------------------------
ERROR HANDLING
-----------------------------------
- If user refuses or doesn't know a field → set value to "unknown"
- If user gives multiple answers → use the most recent

-----------------------------------
TONE
-----------------------------------
Warm, empathetic, professional. Make the passenger feel heard and supported.

-----------------------------------
IMPORTANT
-----------------------------------
You are NOT a general assistant.

-----------------------------------
VOICE CHAT (important)
-----------------------------------
The passenger may speak their message via voice. Treat voice input the same as typed input.
Keep responses concise and avoid long multi-paragraph blocks.

-----------------------------------
BACKEND ACTIONS (optional)
-----------------------------------
If the passenger asks you to perform one of these actions, respond with a single JSON object (and nothing else)
so the server can call the backend:

0) Check the status / see if their lost item was found (if they say they've already filed a report, or ask for updates):
{"action":"check_my_lost_item","data":{"status":"open","limit":10}}

1) List their lost reports:
{"action":"list_lost_reports","data":{"status":""}}

2) Delete a lost report:
{"action":"delete_lost_report","data":{"lost_report_id":""}}

3) Search for matches for a specific lost report:
{"action":"search_found_item_matches","data":{"lost_report_id":"","limit":10}}

4) File a claim on a found item:
{"action":"file_claim","data":{"found_item_id":"","lost_report_id":"","message":""}}

5) List my claims:
{"action":"list_my_claims","data":{"status":"","limit":50,"offset":0}}

For creating a lost report, keep using the existing confirmed intake JSON format."""


class Message(BaseModel):
    role: str
    content: str


class ChatRequest(BaseModel):
    messages: List[Message]
    passenger_id: str | None = None
    forwarded_token: str | None = None


class ChatResponse(BaseModel):
    reply: str
    done: bool
    action: str | None = None
    grpc_ok: bool | None = None
    grpc_data: dict | None = None
    grpc_error: str | None = None


openai_client = OpenAI(api_key=os.environ.get("OPENAI_API_KEY"))

from grpc_handler import PassengerGrpcHandler  # noqa: E402

passenger_grpc = PassengerGrpcHandler()

_SESSION_TTL_SECONDS = 30 * 60
_SESSION_MAX_MESSAGES = 60
_session_lock = asyncio.Lock()
_sessions: dict[str, dict] = {}
_CENTRAL_TZ = ZoneInfo("America/Chicago")
_ITEM_CONTEXT_SAME = "same"
_ITEM_CONTEXT_DIFFERENT = "different"

def _utc_now() -> datetime:
    return datetime.now(timezone.utc)

def _central_now() -> datetime:
    return datetime.now(_CENTRAL_TZ)

def _today_iso() -> str:
    return _central_now().date().isoformat()

def _resolve_relative_date_words(text: str) -> str:
    lower = (text or "").strip().lower()
    today = _central_now().date()
    if "day before yesterday" in lower:
        return (today - timedelta(days=2)).isoformat()
    if "yesterday" in lower:
        return (today - timedelta(days=1)).isoformat()
    if "today" in lower:
        return today.isoformat()
    return ""

def _looks_like_status_check(user_text: str) -> bool:
    t = (user_text or "").strip().lower()
    if not t:
        return False
    phrases = [
        "already filed",
        "already submitted",
        "already made a report",
        "i filed",
        "i filed my report",
        "check if",
        "did you find",
        "have you found",
        "any update",
        "any updates",
        "status of my report",
        "see if you found",
        "see if you've found",
        "found my",
        "matches for my",
    ]
    return any(p in t for p in phrases)

def _looks_like_list_lost_reports(user_text: str) -> bool:
    t = (user_text or "").strip().lower()
    if not t:
        return False
    return ("lost report" in t or "lost reports" in t) and any(w in t for w in ("show", "list", "see", "all"))

def _looks_like_list_claims(user_text: str) -> bool:
    t = (user_text or "").strip().lower()
    if not t:
        return False
    return ("claim" in t or "claims" in t) and any(w in t for w in ("show", "list", "see", "all"))

def _looks_like_delete_lost_report(user_text: str) -> bool:
    t = (user_text or "").strip().lower()
    if not t:
        return False
    return ("delete" in t or "remove" in t) and ("lost report" in t or "report" in t)


# Stable marker used by the "which report?" prompt so we can recognize it in
# the assistant history and resolve the passenger's next answer to a specific
# lost_report_id.
_CHECK_CHOICE_MARKER = "Which report should I check?"


def _format_reports_choice_prompt(reports: list[dict]) -> str:
    lines = []
    for i, r in enumerate(reports, start=1):
        name = (r.get("item_name") or "Unnamed").strip()
        status = (r.get("status") or "open").strip()
        lines.append(f"{i}. {name} — {status}")
    listing = "\n".join(lines)
    return (
        "You have multiple open lost reports. "
        + _CHECK_CHOICE_MARKER
        + "\n\n"
        + listing
        + "\n\n"
        "Reply with the number, the item name, or say \"most recent\" to check the latest."
    )


_UUID_RE = re.compile(
    r"\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b",
    re.IGNORECASE,
)


def _resolve_report_choice(user_text: str, reports: list[dict]) -> str:
    """Given the passenger's reply after a choice prompt, return a report id.

    Returns "" when the reply is vague or ambiguous. The caller may then
    default to the most recent report.
    """
    if not reports:
        return ""
    t = (user_text or "").strip().lower()
    if not t:
        return ""

    # Explicit UUID wins.
    m = _UUID_RE.search(t)
    if m:
        rid = m.group(0)
        for r in reports:
            if str(r.get("id") or "").strip().lower() == rid.lower():
                return str(r.get("id") or "")

    # Numeric index (1-based).
    mnum = re.search(r"\b(\d{1,2})\b", t)
    if mnum:
        try:
            idx = int(mnum.group(1)) - 1
            if 0 <= idx < len(reports):
                return str(reports[idx].get("id") or "")
        except ValueError:
            pass

    # Item-name contains match (case-insensitive).
    for r in reports:
        name = (r.get("item_name") or "").strip().lower()
        if len(name) >= 3 and name in t:
            return str(r.get("id") or "")

    return ""


def _answer_is_vague_choice(user_text: str) -> bool:
    t = (user_text or "").strip().lower()
    if not t:
        return False
    phrases = (
        "most recent",
        "latest",
        "newest",
        "last one",
        "the last",
        "any one",
        "any",
        "either",
        "idk",
        "i don't know",
        "dont know",
        "don't care",
        "doesn't matter",
        "doesnt matter",
        "whatever",
        "you pick",
        "up to you",
    )
    return any(p in t for p in phrases)


def _last_assistant_has_check_choice_prompt(conversation: list[dict]) -> bool:
    for msg in reversed(conversation[:-1]):
        if (msg.get("role") or "") == "assistant":
            return _CHECK_CHOICE_MARKER in str(msg.get("content") or "")
    return False


def _normalize_item_key(text: str) -> str:
    cleaned = re.sub(r"[^a-z0-9]+", " ", (text or "").strip().lower())
    return " ".join(cleaned.split())


def _extract_item_name_from_messages(messages: List[dict]) -> str:
    try:
        state = _extract_slots_from_conversation(messages)
    except Exception:
        return ""
    return str(state.get("item_name") or "").strip()


def _extract_item_name_from_text(text: str) -> str:
    return _extract_item_name_from_messages([{"role": "user", "content": text}])


def _new_context_id(session: dict) -> str:
    seq = int(session.get("next_context_seq") or 1)
    session["next_context_seq"] = seq + 1
    return f"ctx-{seq}"


def _refresh_context_metadata(context: dict, *, now: datetime | None = None) -> None:
    item_name = _extract_item_name_from_messages(context.get("messages") or [])
    context["item_name"] = item_name
    context["item_key"] = _normalize_item_key(item_name)
    context["updated_at"] = now or _utc_now()


def _make_context(*, now: datetime, messages: List[dict] | None = None) -> dict:
    ctx = {
        "messages": list(messages or [])[-_SESSION_MAX_MESSAGES:],
        "item_name": "",
        "item_key": "",
        "updated_at": now,
    }
    _refresh_context_metadata(ctx, now=now)
    return ctx


def _make_session(now: datetime, messages: List[dict] | None = None) -> dict:
    session = {
        "contexts": {},
        "active_context_id": None,
        "pending_item_switch": None,
        "next_context_seq": 1,
        "updated_at": now,
    }
    ctx_id = _new_context_id(session)
    session["contexts"][ctx_id] = _make_context(now=now, messages=messages)
    session["active_context_id"] = ctx_id
    return session


def _ensure_session_shape(raw_session: dict | None, *, now: datetime) -> dict:
    if raw_session is None or (now - raw_session["updated_at"]).total_seconds() > _SESSION_TTL_SECONDS:
        return _make_session(now)

    if "contexts" in raw_session:
        raw_session["updated_at"] = now
        if raw_session.get("active_context_id") not in raw_session.get("contexts", {}):
            contexts = raw_session.get("contexts") or {}
            if contexts:
                raw_session["active_context_id"] = next(iter(contexts))
            else:
                ctx_id = _new_context_id(raw_session)
                raw_session["contexts"][ctx_id] = _make_context(now=now)
                raw_session["active_context_id"] = ctx_id
        return raw_session

    messages = list(raw_session.get("messages") or [])
    return _make_session(now, messages=messages)


def _get_active_context(session: dict, *, now: datetime) -> tuple[str, dict]:
    ctx_id = session.get("active_context_id")
    contexts = session.setdefault("contexts", {})
    if ctx_id and ctx_id in contexts:
        return ctx_id, contexts[ctx_id]
    if contexts:
        first_id = next(iter(contexts))
        session["active_context_id"] = first_id
        return first_id, contexts[first_id]
    new_id = _new_context_id(session)
    contexts[new_id] = _make_context(now=now)
    session["active_context_id"] = new_id
    return new_id, contexts[new_id]


def _find_context_id_by_item_key(session: dict, item_key: str) -> str:
    if not item_key:
        return ""
    for ctx_id, ctx in (session.get("contexts") or {}).items():
        if str(ctx.get("item_key") or "") == item_key:
            return ctx_id
    return ""


def _append_user_message(context: dict, message: dict, *, now: datetime) -> None:
    if (
        not context["messages"]
        or context["messages"][-1].get("role") != "user"
        or context["messages"][-1].get("content") != message.get("content")
    ):
        context["messages"].append(message)
    context["messages"] = context["messages"][-_SESSION_MAX_MESSAGES:]
    _refresh_context_metadata(context, now=now)


def _append_assistant_message(context: dict, reply: str, *, now: datetime) -> None:
    context["messages"].append({"role": "assistant", "content": reply})
    context["messages"] = context["messages"][-_SESSION_MAX_MESSAGES:]
    context["updated_at"] = now


def _resolve_item_switch_decision(user_text: str) -> str:
    t = (user_text or "").strip().lower()
    if not t:
        return ""
    same_phrases = (
        "same report",
        "same item",
        "same one",
        "correcting it",
        "correction",
        "update it",
        "update the report",
        "edit it",
        "still the same",
    )
    different_phrases = (
        "different item",
        "different one",
        "new item",
        "another item",
        "separate item",
        "different report",
        "new report",
    )
    if any(p in t for p in different_phrases):
        return _ITEM_CONTEXT_DIFFERENT
    if any(p in t for p in same_phrases):
        return _ITEM_CONTEXT_SAME
    if re.search(r"\b(same|correct|correction|update|edit)\b", t):
        return _ITEM_CONTEXT_SAME
    if re.search(r"\b(different|another|separate|new)\b", t):
        return _ITEM_CONTEXT_DIFFERENT
    return ""


def _item_switch_prompt(current_item: str, new_item: str) -> str:
    current_label = current_item or "your current item"
    new_label = new_item or "this item"
    return (
        f"I already have a report in progress for {current_label}. "
        f"You just mentioned {new_label}. "
        "Is this the same report and you're correcting the item, or is this a different item? "
        'Reply with "same report" or "different item".'
    )


def _force_check_ambiguity(reply_text: str) -> str:
    """If the LLM emits a check_my_lost_item action without a specific
    lost_report_id, force ``auto_default=False`` so the backend returns a
    ``needs_choice`` signal instead of silently picking the most recent.
    """
    t = (reply_text or "").strip()
    if not (t.startswith("{") and t.endswith("}")):
        return reply_text
    try:
        obj = json.loads(t)
    except Exception:
        return reply_text
    if not isinstance(obj, dict):
        return reply_text
    action = str(obj.get("action") or obj.get("intent") or "").strip().lower()
    if action.replace("-", "_") != "check_my_lost_item":
        return reply_text
    data = obj.get("data") if isinstance(obj.get("data"), dict) else {}
    if (str(data.get("lost_report_id") or "")).strip():
        return reply_text
    data["auto_default"] = False
    obj["data"] = data
    return json.dumps(obj)

_COLOR_WORDS = {
    "black",
    "white",
    "gray",
    "grey",
    "silver",
    "gold",
    "red",
    "blue",
    "green",
    "yellow",
    "orange",
    "purple",
    "pink",
    "brown",
    "beige",
    "tan",
    "navy",
}

_BRANDS = {
    "apple",
    "samsung",
    "google",
    "sony",
    "bose",
    "dell",
    "hp",
    "lenovo",
    "asus",
    "nike",
    "adidas",
}

_ITEM_KEYWORDS = {
    "airpod": "AirPods",
    "airpods": "AirPods",
    "earpod": "EarPods",
    "earpods": "EarPods",
    "wallet": "Wallet",
    "phone": "Phone",
    "iphone": "iPhone",
    "android": "Phone",
    "backpack": "Backpack",
    "bag": "Bag",
    "laptop": "Laptop",
    "keys": "Keys",
    "key": "Keys",
    "headphones": "Headphones",
    "earbuds": "Earbuds",
    "glasses": "Glasses",
}

_SLOT_ORDER = [
    "item_name",
    "color",
    "brand",
    "description",
    "route_from",
    "route_to",
    "date_lost",
    "time_lost",
]

def _clean_route_value(text: str) -> str:
    cleaned = re.sub(r"^[\s,.;:!?-]+|[\s,.;:!?-]+$", "", text or "")
    cleaned = re.sub(r"\s+", " ", cleaned).strip()
    if not cleaned:
        return ""
    # Trim leading phrasing so we keep only city/stop text.
    cleaned = re.sub(
        r"^(?:i(?:'m| am)?\s+)?(?:was\s+)?(?:going|headed|heading|travel(?:ling)?|coming|from|to)\s+",
        "",
        cleaned,
        flags=re.IGNORECASE,
    ).strip()
    # Keep up to two words for compact city/stop names.
    tokens = cleaned.split()
    if len(tokens) > 2:
        cleaned = " ".join(tokens[:2])
    return cleaned

_UNKNOWN_ANSWER_PATTERNS = [
    r"^\s*unknown\b",
    r"\bunknown\s+(?:brand|color|colour|model|type|kind)\b",
    r"^\s*(?:i|we)?\s*(?:do\s*n'?t|don'?t|do not)\s+(?:know|remember|recall)\b",
    r"^\s*not\s+(?:sure|really sure)\b",
    r"^\s*no\s+idea\b",
    r"^\s*n\s*/?\s*a\s*$",
    r"^\s*none\s*$",
    r"^\s*skip\s*$",
]

def _is_unknown_answer(text: str) -> bool:
    t = (text or "").strip().lower()
    if not t:
        return False
    return any(re.search(p, t) for p in _UNKNOWN_ANSWER_PATTERNS)

def _short_clean(text: str, max_words: int) -> str:
    cleaned = re.sub(r"^[\s,.;:!?-]+|[\s,.;:!?-]+$", "", text or "")
    cleaned = re.sub(r"\s+", " ", cleaned).strip()
    if not cleaned:
        return ""
    tokens = cleaned.split()
    if len(tokens) > max_words:
        cleaned = " ".join(tokens[:max_words])
    return cleaned

def _parse_route_pair(text: str) -> tuple[str, str]:
    msg = (text or "").strip()
    if not msg:
        return "", ""
    m = re.search(
        r"\bfrom\s+([a-zA-Z][a-zA-Z ]{0,40}?)\s+to\s+([a-zA-Z][a-zA-Z ]{0,40}?)\b",
        msg,
        re.IGNORECASE,
    )
    if m:
        return _clean_route_value(m.group(1)), _clean_route_value(m.group(2))
    m = re.search(
        r"\b([a-zA-Z][a-zA-Z ]{0,40}?)\s+to\s+([a-zA-Z][a-zA-Z ]{0,40}?)\b",
        msg,
        re.IGNORECASE,
    )
    if m:
        return _clean_route_value(m.group(1)), _clean_route_value(m.group(2))
    return "", ""

def _parse_time_str(text: str) -> str:
    """Parse any reasonable time expression into 24-hour HH:MM. Returns '' if unparseable."""
    t = (text or "").strip().lower()
    if not t:
        return ""
    if "noon" in t:
        return "12:00"
    if "midnight" in t:
        return "00:00"
    # HH:MM with optional am/pm — e.g. "13:00", "1:30 pm"
    m = re.search(r"\b(\d{1,2}):(\d{2})\s*([ap]\.?m\.?)?\b", t, re.IGNORECASE)
    if m:
        hour, minute = int(m.group(1)), int(m.group(2))
        period = (m.group(3) or "").lower().replace(".", "")
        if period == "pm" and hour != 12:
            hour += 12
        elif period == "am" and hour == 12:
            hour = 0
        return f"{hour:02d}:{minute:02d}"
    # Hour-only with am/pm — e.g. "1pm", "2 AM", "at 1PM"
    m = re.search(r"\b(\d{1,2})\s*([ap]\.?m\.?)\b", t, re.IGNORECASE)
    if m:
        hour = int(m.group(1))
        period = m.group(2).lower().replace(".", "")
        if period == "pm" and hour != 12:
            hour += 12
        elif period == "am" and hour == 12:
            hour = 0
        return f"{hour:02d}:00"
    return ""


def _extract_slots_from_conversation(conversation: List[dict]) -> dict:
    state = {k: "" for k in _SLOT_ORDER}
    user_texts = [str(m.get("content") or "") for m in conversation if m.get("role") == "user"]
    combined = " \n".join(user_texts)
    lower_all = combined.lower()

    # item_name
    mver = re.search(r"\bairpods?\s*(\d)\b", lower_all)
    if mver:
        state["item_name"] = f"AirPods {mver.group(1)}"
    for k, v in _ITEM_KEYWORDS.items():
        if not state["item_name"] and re.search(rf"\b{re.escape(k)}\b", lower_all):
            state["item_name"] = v
            break
    if not state["item_name"]:
        m = re.search(r"\blost (?:my|a|an|the)\s+([a-zA-Z0-9][^.,\n]{0,40})", lower_all)
        if m:
            state["item_name"] = m.group(1).strip()[:40]

    # brand
    for b in _BRANDS:
        if re.search(rf"\b{re.escape(b)}\b", lower_all):
            state["brand"] = b.title()
            break

    # color
    for c in _COLOR_WORDS:
        if re.search(rf"\b{re.escape(c)}\b", lower_all):
            state["color"] = c
            break

    # description (grab last relevant message)
    for t in reversed(user_texts):
        tl = t.lower()
        if any(w in tl for w in (
            "logo",
            "sticker",
            "scratch",
            "crack",
            "case",
            "engrave",
            "initial",
            "leather",
            "pattern",
            "plain",
            "nothing",
            "no distinguishing",
            "no distinguishing features",
        )):
            state["description"] = t.strip()[:160]
            break

    # Prefer direct answers to the slot the assistant just asked about.
    for idx in range(1, len(conversation)):
        msg = conversation[idx]
        prev = conversation[idx - 1]
        if msg.get("role") != "user" or prev.get("role") != "assistant":
            continue
        asked = _asked_slot_from_reply(str(prev.get("content") or ""))
        if not asked:
            continue
        user_msg = str(msg.get("content") or "").strip()
        if not user_msg:
            continue

        if asked in ("route_from", "route_to"):
            frm, to = _parse_route_pair(user_msg)
            if frm and not state["route_from"]:
                state["route_from"] = frm
            if to and not state["route_to"]:
                state["route_to"] = to
            if asked == "route_from" and not state["route_from"]:
                state["route_from"] = _clean_route_value(user_msg)
            elif asked == "route_to" and not state["route_to"]:
                state["route_to"] = _clean_route_value(user_msg)
            continue

        if state.get(asked):
            continue

        if _is_unknown_answer(user_msg):
            state[asked] = "unknown"
            continue

        if asked == "item_name":
            state[asked] = _short_clean(user_msg, 5)[:40]
        elif asked == "color":
            state[asked] = _short_clean(user_msg, 3)[:40]
        elif asked == "brand":
            state[asked] = _short_clean(user_msg, 4)[:40]
        elif asked == "description":
            state[asked] = user_msg[:160]
        elif asked == "date_lost":
            mdate = re.search(r"\b(20\d{2}-\d{2}-\d{2})\b", user_msg)
            if mdate:
                state[asked] = mdate.group(1)
            else:
                rel = _resolve_relative_date_words(user_msg.lower())
                if rel:
                    state[asked] = rel
        elif asked == "time_lost":
            state[asked] = _parse_time_str(user_msg)

    frm, to = _parse_route_pair(combined)
    if not state["route_from"] and frm:
        state["route_from"] = frm
    if not state["route_to"] and to:
        state["route_to"] = to
    if not state["route_from"]:
        m2 = re.search(r"\bat\s+([a-zA-Z][a-zA-Z ]{0,40})\b", combined, re.IGNORECASE)
        if m2:
            state["route_from"] = _clean_route_value(m2.group(1))
    if not state["route_to"]:
        m3 = re.search(r"\b(?:going|heading|traveling|travelling)\s+to\s+([a-zA-Z][a-zA-Z ]{0,40})\b", combined, re.IGNORECASE)
        if m3:
            state["route_to"] = _clean_route_value(m3.group(1))

    # date/time if explicit
    mdate = re.search(r"\b(20\d{2}-\d{2}-\d{2})\b", lower_all)
    if mdate:
        state["date_lost"] = mdate.group(1)
    if not state["date_lost"]:
        rel = _resolve_relative_date_words(lower_all)
        if rel:
            state["date_lost"] = rel
    parsed_time = _parse_time_str(lower_all)
    if parsed_time:
        state["time_lost"] = parsed_time

    for k in _SLOT_ORDER:
        state[k] = str(state.get(k) or "").strip()
    return state

def _missing_slots(state: dict) -> list[str]:
    return [k for k in _SLOT_ORDER if not str(state.get(k) or "").strip()]

def _looks_like_intake_confirmation(user_text: str) -> bool:
    t = (user_text or "").strip().lower()
    if not t:
        return False
    if re.search(r"\b(yes|yep|yeah|correct|confirmed|confirm)\b", t):
        return True
    if "everything is correct" in t or "looks correct" in t:
        return True
    if "submit" in t or "file it" in t or "go ahead" in t:
        return True
    return False

def _final_intake_json_from_state(state: dict) -> str:
    payload = {k: str(state.get(k) or "").strip() for k in _SLOT_ORDER}
    return json.dumps(payload)

def _slot_question(slot: str) -> str:
    if slot == "item_name":
        return "What item did you lose?"
    if slot == "color":
        return "What color was it?"
    if slot == "brand":
        return "What brand is it (if you know)?"
    if slot == "description":
        return "Any distinguishing details (stickers, scratches, case, contents, etc.)?"
    if slot == "route_from":
        return "What was your departure city or stop?"
    if slot == "route_to":
        return "What was your destination city or stop?"
    if slot == "date_lost":
        return "What date did you lose it? (YYYY-MM-DD if possible)"
    if slot == "time_lost":
        return "About what time did you lose it? (e.g., 2:30 PM)"
    return "Could you tell me a bit more?"

def _asked_slot_from_reply(reply: str) -> str:
    t = (reply or "").strip().lower()
    if not t:
        return ""
    if "what item" in t or "what did you lose" in t:
        return "item_name"
    if "what color" in t or "colour" in t:
        return "color"
    if "brand" in t:
        return "brand"
    if "distinguishing" in t or "description" in t or "details" in t or "features" in t:
        return "description"
    if "departure" in t:
        return "route_from"
    if "destination" in t:
        return "route_to"
    if "date" in t:
        return "date_lost"
    if "time" in t:
        return "time_lost"
    return ""

def _maybe_override_reply(reply: str, conversation: List[dict]) -> str | None:
    # Don't override backend action JSON or intake JSON.
    rt = (reply or "").strip()
    if rt.startswith("{") and rt.endswith("}"):
        try:
            obj = json.loads(rt)
            if isinstance(obj, dict) and (obj.get("action") or obj.get("intent")):
                return None
            if isinstance(obj, dict) and any(k in obj for k in ("item_name", "color", "brand", "description", "route_from", "route_to", "date_lost", "time_lost")):
                return None
        except Exception:
            pass

    state = _extract_slots_from_conversation(conversation)
    missing = _missing_slots(state)
    if not missing:
        return None

    asked = _asked_slot_from_reply(reply)
    if asked and state.get(asked):
        return _slot_question(missing[0])

    # Script restart protection.
    if rt.lower().startswith("hello!") and any(state.get(k) for k in ("item_name", "color", "brand", "description", "route_from", "route_to")):
        return _slot_question(missing[0])

    return None


def call_openai(conversation: List[dict]) -> str:
    messages = [{"role": "system", "content": SYSTEM_PROMPT}] + conversation
    try:
        state = _extract_slots_from_conversation(conversation)
        missing = _missing_slots(state)
        messages.append(
            {
                "role": "system",
                "content": f"Current date (Central Time) is {_today_iso()}. Use this to resolve words like today/yesterday.\n"
                + "Known fields so far (do not ask again for these): "
                + json.dumps(state)
                + "\nMissing fields (ask ONLY for the next one): "
                + json.dumps(missing[:1]),
            }
        )
    except Exception:
        pass
    model = (os.environ.get("OPENAI_CHAT_MODEL") or "gpt-4o-mini").strip()
    completion = openai_client.chat.completions.create(
        model=model,
        messages=messages,
        temperature=0.3,
        max_tokens=512,
    )
    return completion.choices[0].message.content


def is_done(reply: str) -> bool:
    t = reply.strip()
    return t.startswith("{") and t.endswith("}")

async def _session_conversation_for_request(
    passenger_id: str, incoming: List[dict]
) -> tuple[List[dict], str | None]:
    now = _utc_now()
    async with _session_lock:
        session = _ensure_session_shape(_sessions.get(passenger_id), now=now)
        _sessions[passenger_id] = session

        active_id, active_ctx = _get_active_context(session, now=now)

        # If the client sent a substantial history, treat it as canonical for the active item context.
        if len(incoming) >= 6:
            active_ctx["messages"] = incoming[-_SESSION_MAX_MESSAGES:]
            _refresh_context_metadata(active_ctx, now=now)
            session["updated_at"] = now
            return list(active_ctx["messages"]), None

        last_user = None
        for m in reversed(incoming):
            if (m.get("role") or "").strip() == "user":
                last_user = {"role": "user", "content": str(m.get("content") or "")}
                break
        if last_user is None:
            session["updated_at"] = now
            return list(active_ctx["messages"]), None

        pending = session.get("pending_item_switch")
        if pending:
            decision = _resolve_item_switch_decision(last_user["content"])
            if decision == _ITEM_CONTEXT_SAME:
                source_id = pending.get("source_context_id") or active_id
                source_ctx = session["contexts"].get(source_id, active_ctx)
                _append_user_message(source_ctx, pending["message"], now=now)
                session["active_context_id"] = source_id
                session["pending_item_switch"] = None
                session["updated_at"] = now
                return list(source_ctx["messages"]), None

            if decision == _ITEM_CONTEXT_DIFFERENT:
                proposed_key = str(pending.get("proposed_item_key") or "")
                target_id = _find_context_id_by_item_key(session, proposed_key)
                if target_id:
                    target_ctx = session["contexts"][target_id]
                else:
                    target_id = _new_context_id(session)
                    target_ctx = _make_context(now=now)
                    session["contexts"][target_id] = target_ctx
                _append_user_message(target_ctx, pending["message"], now=now)
                session["active_context_id"] = target_id
                session["pending_item_switch"] = None
                session["updated_at"] = now
                return list(target_ctx["messages"]), None

            session["updated_at"] = now
            return list(active_ctx["messages"]), _item_switch_prompt(
                str(pending.get("current_item_name") or ""),
                str(pending.get("proposed_item_name") or ""),
            )

        proposed_item_name = _extract_item_name_from_text(last_user["content"])
        proposed_item_key = _normalize_item_key(proposed_item_name)
        current_item_name = str(active_ctx.get("item_name") or "")
        current_item_key = str(active_ctx.get("item_key") or "")

        if proposed_item_key:
            if current_item_key and proposed_item_key != current_item_key:
                session["pending_item_switch"] = {
                    "source_context_id": active_id,
                    "message": last_user,
                    "current_item_name": current_item_name,
                    "proposed_item_name": proposed_item_name,
                    "proposed_item_key": proposed_item_key,
                }
                session["updated_at"] = now
                return list(active_ctx["messages"]), _item_switch_prompt(
                    current_item_name,
                    proposed_item_name,
                )

            existing_id = _find_context_id_by_item_key(session, proposed_item_key)
            if existing_id and existing_id != active_id:
                target_ctx = session["contexts"][existing_id]
                _append_user_message(target_ctx, last_user, now=now)
                session["active_context_id"] = existing_id
                session["updated_at"] = now
                return list(target_ctx["messages"]), None

        _append_user_message(active_ctx, last_user, now=now)
        session["updated_at"] = now
        return list(active_ctx["messages"]), None

async def _session_append_assistant(passenger_id: str, reply: str) -> None:
    now = _utc_now()
    async with _session_lock:
        session = _ensure_session_shape(_sessions.get(passenger_id), now=now)
        _sessions[passenger_id] = session
        _, active_ctx = _get_active_context(session, now=now)
        _append_assistant_message(active_ctx, reply, now=now)
        session["updated_at"] = now


@app.get("/health")
def health():
    return "Chat bot is running!!!"


@app.post("/chat", response_model=ChatResponse)
async def chat(req: ChatRequest):
    if not os.environ.get("OPENAI_API_KEY"):
        raise HTTPException(status_code=500, detail="OPENAI_API_KEY not configured")
    try:
        incoming = [{"role": m.role, "content": m.content} for m in req.messages]
        conversation = incoming
        if req.passenger_id:
            conversation, immediate_reply = await _session_conversation_for_request(
                req.passenger_id, incoming
            )
            if immediate_reply:
                await _session_append_assistant(req.passenger_id, immediate_reply)
                return ChatResponse(reply=immediate_reply, done=False)
        # Deterministic confirmation handoff: when all intake slots are present
        # and the user confirms/asks to submit, bypass potential LLM drift and
        # emit the final intake JSON directly.
        reply = ""
        try:
            state = _extract_slots_from_conversation(conversation)
            missing = _missing_slots(state)
            last_user_text = ""
            for m in reversed(conversation):
                if (m.get("role") or "").strip() == "user":
                    last_user_text = str(m.get("content") or "")
                    break
            if (not missing) and _looks_like_intake_confirmation(last_user_text):
                reply = _final_intake_json_from_state(state)
            else:
                reply = await asyncio.to_thread(call_openai, conversation)
        except Exception:
            reply = await asyncio.to_thread(call_openai, conversation)
        try:
            override = _maybe_override_reply(reply, conversation)
            if override:
                reply = override
        except Exception:
            pass
        dispatch = None
        if req.passenger_id:
            # If the previous assistant turn asked which lost report to check,
            # resolve the passenger's reply deterministically instead of going
            # through the LLM again.
            if conversation and _last_assistant_has_check_choice_prompt(conversation):
                last_user_text = str(conversation[-1].get("content", "") or "")
                list_resp = await passenger_grpc.list_lost_reports(
                    passenger_id=req.passenger_id,
                    status="open",
                    forwarded_token=req.forwarded_token,
                )
                list_dict = MessageToDict(list_resp, preserving_proto_field_name=True)
                open_reports = list_dict.get("reports") or []
                chosen_id = _resolve_report_choice(last_user_text, open_reports)
                if not chosen_id and open_reports and _answer_is_vague_choice(last_user_text):
                    chosen_id = str(open_reports[0].get("id") or "")
                if chosen_id:
                    dispatch = await passenger_grpc.dispatch_from_chat_reply(
                        passenger_id=req.passenger_id,
                        chat_reply_text=json.dumps(
                            {
                                "action": "check_my_lost_item",
                                "data": {
                                    "status": "open",
                                    "limit": 10,
                                    "lost_report_id": chosen_id,
                                    "auto_default": True,
                                },
                            }
                        ),
                        forwarded_token=req.forwarded_token,
                    )
                    reply = "Got it — checking that report for matching found items…"

            if dispatch is None:
                await _session_append_assistant(req.passenger_id, reply)
                dispatch = await passenger_grpc.dispatch_from_chat_reply(
                    passenger_id=req.passenger_id,
                    chat_reply_text=_force_check_ambiguity(reply),
                    forwarded_token=req.forwarded_token,
                )
            # Fallback: if the model failed to trigger the backend check, do it automatically.
            if dispatch is not None and dispatch.action == "none" and conversation:
                last_user = conversation[-1].get("content", "")
                if _looks_like_status_check(last_user):
                    dispatch = await passenger_grpc.dispatch_from_chat_reply(
                        passenger_id=req.passenger_id,
                        chat_reply_text='{"action":"check_my_lost_item","data":{"status":"open","limit":10,"auto_default":false}}',
                        forwarded_token=req.forwarded_token,
                    )
                    reply = "Let me check your existing report for any matching found items…"
                elif _looks_like_list_claims(last_user):
                    dispatch = await passenger_grpc.dispatch_from_chat_reply(
                        passenger_id=req.passenger_id,
                        chat_reply_text='{"action":"list_my_claims","data":{"status":"","limit":50,"offset":0}}',
                        forwarded_token=req.forwarded_token,
                    )
                    reply = "Sure — here are your claims."
                elif _looks_like_list_lost_reports(last_user):
                    dispatch = await passenger_grpc.dispatch_from_chat_reply(
                        passenger_id=req.passenger_id,
                        chat_reply_text='{"action":"list_lost_reports","data":{"status":""}}',
                        forwarded_token=req.forwarded_token,
                    )
                    reply = "Sure — here are your lost reports."
                elif _looks_like_delete_lost_report(last_user):
                    dispatch = await passenger_grpc.dispatch_from_chat_reply(
                        passenger_id=req.passenger_id,
                        chat_reply_text='{"action":"list_lost_reports","data":{"status":""}}',
                        forwarded_token=req.forwarded_token,
                    )
                    reply = "I can help delete a report. Which one would you like to delete?"

        if dispatch is None:
            return ChatResponse(reply=reply, done=is_done(reply))

        # If a check_my_lost_item dispatch came back needing disambiguation,
        # first try to resolve the ambiguity from the user's original query
        # (e.g. "find my keys" already names the item → no prompt needed).
        # Only fall back to the choice prompt when the original message is
        # genuinely ambiguous.
        if (
            dispatch.action == "check_my_lost_item"
            and dispatch.ok
            and isinstance(dispatch.data, dict)
            and dispatch.data.get("needs_choice")
        ):
            reports_list = dispatch.data.get("reports") or []
            # Attempt to resolve from the original user message that triggered
            # this path, before forcing a follow-up question.
            original_user_text = str(conversation[-1].get("content", "") or "") if conversation else ""
            pre_chosen = _resolve_report_choice(original_user_text, reports_list)
            if not pre_chosen and reports_list and _answer_is_vague_choice(original_user_text):
                pre_chosen = str(reports_list[0].get("id") or "")

            if pre_chosen:
                # We resolved it — dispatch directly without prompting.
                dispatch = await passenger_grpc.dispatch_from_chat_reply(
                    passenger_id=req.passenger_id,
                    chat_reply_text=json.dumps(
                        {
                            "action": "check_my_lost_item",
                            "data": {
                                "status": "open",
                                "limit": 10,
                                "lost_report_id": pre_chosen,
                                "auto_default": True,
                            },
                        }
                    ),
                    forwarded_token=req.forwarded_token,
                )
                reply = "Got it — checking that report for matching found items…"
            else:
                prompt_reply = _format_reports_choice_prompt(reports_list)
                if req.passenger_id:
                    try:
                        await _session_append_assistant(req.passenger_id, prompt_reply)
                    except Exception:
                        pass
                return ChatResponse(reply=prompt_reply, done=False)

        return ChatResponse(
            reply=reply,
            done=is_done(reply),
            action=dispatch.action,
            grpc_ok=dispatch.ok,
            grpc_data=dispatch.data,
            grpc_error=dispatch.error,
        )
    except Exception as e:
        logger.error(f"Chat error: {e}", exc_info=True)
        # Do not leak raw internal errors to end-users.
        raise HTTPException(status_code=500, detail="Chat agent error")

class TTSRequest(BaseModel):
    text: str

class TTSResponse(BaseModel):
    mime: str
    audio_base64: str


class ClaimRequest(BaseModel):
    passenger_id: str
    found_item_id: str
    lost_report_id: str
    message: str | None = None
    forwarded_token: str | None = None


class ClaimResponse(BaseModel):
    ok: bool
    data: dict | None = None
    error: str | None = None


@app.post("/claim", response_model=ClaimResponse)
async def file_claim_direct(req: ClaimRequest):
    """Deterministic claim endpoint that bypasses the LLM.

    Takes a passenger_id + found_item_id + lost_report_id and calls the backend
    FileClaim RPC directly. Used by the match-card 'File claim' button so the
    claim action isn't routed back through the model.
    """
    passenger_id = (req.passenger_id or "").strip()
    found_item_id = (req.found_item_id or "").strip()
    lost_report_id = (req.lost_report_id or "").strip()
    if not passenger_id or not found_item_id or not lost_report_id:
        raise HTTPException(
            status_code=400,
            detail="passenger_id, found_item_id and lost_report_id are required",
        )
    payload = {
        "action": "file_claim",
        "data": {
            "found_item_id": found_item_id,
            "lost_report_id": lost_report_id,
            "message": (req.message or "I believe this is my item.").strip(),
        },
    }
    try:
        dispatch = await passenger_grpc.dispatch_from_chat_reply(
            passenger_id=passenger_id,
            chat_reply_text=json.dumps(payload),
            forwarded_token=req.forwarded_token,
        )
    except Exception as e:
        logger.error(f"Direct file_claim error: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail="file_claim failed")

    if dispatch is None:
        return ClaimResponse(ok=False, error="dispatch returned no result")
    return ClaimResponse(ok=dispatch.ok, data=dispatch.data, error=dispatch.error)


def _deepgram_speak_url() -> str:
    # Default Deepgram Aura voice model.
    model = (os.environ.get("DEEPGRAM_TTS_MODEL") or "aura-asteria-en").strip()
    return f"https://api.deepgram.com/v1/speak?model={model}&encoding=mp3"

async def _deepgram_tts(text: str, api_key: str) -> TTSResponse:
    t = (text or "").strip()
    if not t:
        raise ValueError("text is required")
    # Keep TTS payload bounded.
    if len(t) > 800:
        t = t[:800]

    def _do() -> TTSResponse:
        headers = {
            "Authorization": f"Token {api_key}",
            "Content-Type": "application/json",
            "Accept": "audio/mpeg",
        }
        r = httpx.post(_deepgram_speak_url(), headers=headers, json={"text": t}, timeout=20.0)
        r.raise_for_status()
        mime = (r.headers.get("content-type") or "audio/mpeg").split(";")[0].strip() or "audio/mpeg"
        b64 = base64.b64encode(r.content).decode("ascii")
        return TTSResponse(mime=mime, audio_base64=b64)

    return await asyncio.to_thread(_do)

@app.post("/tts", response_model=TTSResponse)
async def tts(req: TTSRequest):
    api_key = (os.environ.get("DEEPGRAM_API_KEY") or "").strip()
    if not api_key:
        raise HTTPException(status_code=500, detail="DEEPGRAM_API_KEY not configured")
    try:
        return await _deepgram_tts(req.text, api_key)
    except Exception as e:
        logger.error(f"TTS error: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail="TTS failed")


def _deepgram_listen_url() -> str:
    # Browser MediaRecorder chunks are typically Opus in a WebM container.
    # Deepgram supports streaming from browsers; keep params minimal and enable interim results.
    base = "wss://api.deepgram.com/v1/listen"
    params = [
        "punctuate=true",
        "smart_format=true",
        "interim_results=true",
        "endpointing=150",
        "language=en-US",
        "model=nova-2",
        "encoding=opus",
        "container=webm",
    ]
    return f"{base}?{'&'.join(params)}"


@app.websocket("/voice")
async def websocket_voice(websocket: WebSocket):
    """
    Accepts audio chunks from the browser and streams them to Deepgram for STT.
    Emits JSON messages back to the browser:
      {"type":"transcript","text":"..."} (interim)
      {"type":"final","text":"..."} (final transcript)
      {"type":"error"} on failure
    """
    await websocket.accept()

    api_key = (os.environ.get("DEEPGRAM_API_KEY") or "").strip()
    if not api_key:
        await websocket.send_text(json.dumps({"type": "error", "reason": "not_configured"}))
        await websocket.close()
        return

    dg_ws = None
    final_parts: list[str] = []
    stop_requested = asyncio.Event()

    try:
        dg_ws = await websockets.connect(
            _deepgram_listen_url(),
            additional_headers={"Authorization": f"Token {api_key}"},
            ping_interval=20,
            ping_timeout=20,
            max_size=10 * 1024 * 1024,
        )

        async def pump_audio() -> None:
            while True:
                msg = await websocket.receive()
                if msg.get("type") == "websocket.disconnect":
                    stop_requested.set()
                    break
                if msg.get("text") is not None:
                    if (msg.get("text") or "").strip() == "__STOP__":
                        stop_requested.set()
                        break
                    continue
                chunk = msg.get("bytes")
                if chunk:
                    await dg_ws.send(chunk)

        async def pump_transcripts() -> None:
            async for raw in dg_ws:
                try:
                    payload = json.loads(raw)
                except Exception:
                    continue
                # Deepgram sends transcript under channel.alternatives[0].transcript
                channel = payload.get("channel") or {}
                alts = channel.get("alternatives") or []
                transcript = ""
                if alts and isinstance(alts, list):
                    transcript = (alts[0].get("transcript") or "").strip()
                if not transcript:
                    continue
                is_final = bool(payload.get("is_final"))
                speech_final = bool(payload.get("speech_final"))
                if is_final:
                    final_parts.append(transcript)
                    await websocket.send_text(
                        json.dumps({"type": "transcript", "text": " ".join(final_parts)})
                    )
                else:
                    await websocket.send_text(json.dumps({"type": "transcript", "text": transcript}))

                if speech_final and final_parts:
                    utterance = " ".join(final_parts).strip()
                    final_parts.clear()
                    if utterance:
                        await websocket.send_text(json.dumps({"type": "final", "text": utterance}))

        t_audio = asyncio.create_task(pump_audio())
        t_text = asyncio.create_task(pump_transcripts())
        t_text.add_done_callback(lambda _t: stop_requested.set())
        await stop_requested.wait()

        try:
            await dg_ws.send(json.dumps({"type": "CloseStream"}))
        except Exception:
            pass

        try:
            await asyncio.wait_for(t_text, timeout=2.0)
        except Exception:
            t_text.cancel()

        # Flush any partial final transcript on stop.
        final_text = " ".join(final_parts).strip()
        if final_text:
            await websocket.send_text(json.dumps({"type": "final", "text": final_text}))
    except Exception as e:
        logger.error(f"Voice STT error: {e}", exc_info=True)
        try:
            await websocket.send_text(json.dumps({"type": "error", "reason": "stt_failed"}))
        except Exception:
            pass
    finally:
        try:
            if dg_ws is not None:
                await dg_ws.close()
        except Exception:
            pass
        try:
            await websocket.close()
        except Exception:
            pass


@app.websocket("/ws/chat")
async def websocket_chat(websocket: WebSocket):
    await websocket.accept()
    conversation: List[dict] = []
    passenger_id: str | None = None
    forwarded_token: str | None = None

    try:
        while True:
            data = await websocket.receive_text()
            try:
                msg = json.loads(data)
                user_text = msg.get("content", "").strip()
                if msg.get("passenger_id"):
                    passenger_id = str(msg.get("passenger_id")).strip() or passenger_id
                if msg.get("forwarded_token"):
                    forwarded_token = str(msg.get("forwarded_token")).strip() or forwarded_token
            except json.JSONDecodeError:
                user_text = data.strip()

            if not user_text:
                continue

            conversation.append({"role": "user", "content": user_text})

            try:
                reply = call_openai(conversation)
            except Exception as e:
                logger.error(f"Chat websocket error: {e}", exc_info=True)
                reply = (
                    "Sorry — something went wrong while generating my reply. "
                    "Please try again."
                )
            try:
                override = _maybe_override_reply(reply, conversation)
                if override:
                    reply = override
            except Exception:
                pass

            dispatch = None
            if passenger_id:
                dispatch = await passenger_grpc.dispatch_from_chat_reply(
                    passenger_id=passenger_id,
                    chat_reply_text=reply,
                    forwarded_token=forwarded_token,
                )
                if dispatch is not None and dispatch.action == "none" and _looks_like_status_check(user_text):
                    dispatch = await passenger_grpc.dispatch_from_chat_reply(
                        passenger_id=passenger_id,
                        chat_reply_text='{"action":"check_my_lost_item","data":{"status":"open","limit":10}}',
                        forwarded_token=forwarded_token,
                    )
                    reply = "Let me check your existing report for any matching found items…"

            conversation.append({"role": "assistant", "content": reply})

            resp = {"reply": reply, "done": is_done(reply)}
            if dispatch is not None and dispatch.action != "none":
                resp.update(
                    {
                        "action": dispatch.action,
                        "grpc_ok": dispatch.ok,
                        "grpc_data": dispatch.data,
                        "grpc_error": dispatch.error,
                    }
                )

            await websocket.send_text(json.dumps(resp))

    except WebSocketDisconnect:
        pass


@app.on_event("shutdown")
async def _shutdown() -> None:
    await passenger_grpc.close()
