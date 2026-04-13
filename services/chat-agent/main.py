from fastapi import FastAPI, WebSocket, WebSocketDisconnect, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from typing import List
import os
import json
import asyncio
import logging
from dotenv import load_dotenv, find_dotenv
from groq import Groq

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
BACKEND ACTIONS (optional)
-----------------------------------
If the passenger asks you to perform one of these actions, respond with a single JSON object (and nothing else)
so the server can call the backend:

1) List their lost reports:
{"action":"list_lost_reports","data":{"status":""}}

2) Delete a lost report:
{"action":"delete_lost_report","data":{"lost_report_id":""}}

3) Search for matches for a specific lost report:
{"action":"search_found_item_matches","data":{"lost_report_id":"","limit":10}}

4) File a claim on a found item:
{"action":"file_claim","data":{"found_item_id":"","lost_report_id":"","message":""}}

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


client = Groq(api_key=os.environ.get("GROQ_API_KEY"))

from grpc_handler import PassengerGrpcHandler  # noqa: E402

passenger_grpc = PassengerGrpcHandler()


def call_groq(conversation: List[dict]) -> str:
    messages = [{"role": "system", "content": SYSTEM_PROMPT}] + conversation
    completion = client.chat.completions.create(
        model="llama-3.3-70b-versatile",
        messages=messages,
        temperature=0.3,
        max_tokens=512,
    )
    return completion.choices[0].message.content


def is_done(reply: str) -> bool:
    t = reply.strip()
    return t.startswith("{") and t.endswith("}")


@app.get("/health")
def health():
    return "Chat bot is running!!!"


@app.post("/chat", response_model=ChatResponse)
async def chat(req: ChatRequest):
    if not os.environ.get("GROQ_API_KEY"):
        raise HTTPException(status_code=500, detail="GROQ_API_KEY not configured")
    try:
        conversation = [{"role": m.role, "content": m.content} for m in req.messages]
        reply = await asyncio.to_thread(call_groq, conversation)
        dispatch = None
        if req.passenger_id:
            dispatch = await passenger_grpc.dispatch_from_chat_reply(
                passenger_id=req.passenger_id,
                chat_reply_text=reply,
                forwarded_token=req.forwarded_token,
            )

        if dispatch is None:
            return ChatResponse(reply=reply, done=is_done(reply))

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
        raise HTTPException(status_code=500, detail=str(e))


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
                reply = call_groq(conversation)
            except Exception as e:
                reply = f"Error: {str(e)}"

            conversation.append({"role": "assistant", "content": reply})

            dispatch = None
            if passenger_id:
                dispatch = await passenger_grpc.dispatch_from_chat_reply(
                    passenger_id=passenger_id,
                    chat_reply_text=reply,
                    forwarded_token=forwarded_token,
                )

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
