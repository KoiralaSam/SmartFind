from fastapi import FastAPI, WebSocket, WebSocketDisconnect, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from typing import List
import os
import json
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
- date_lost
- time_lost       (approximate is fine)

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
  "date_lost": "",
  "time_lost": ""
}

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
You are NOT a general assistant. Stay focused on the lost item intake task only."""


class Message(BaseModel):
    role: str
    content: str


class ChatRequest(BaseModel):
    messages: List[Message]


class ChatResponse(BaseModel):
    reply: str
    done: bool


client = Groq(api_key=os.environ.get("GROQ_API_KEY"))


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
def chat(req: ChatRequest):
    if not os.environ.get("GROQ_API_KEY"):
        raise HTTPException(status_code=500, detail="GROQ_API_KEY not configured")
    try:
        conversation = [{"role": m.role, "content": m.content} for m in req.messages]
        reply = call_groq(conversation)
        return ChatResponse(reply=reply, done=is_done(reply))
    except Exception as e:
        logger.error(f"Chat error: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=str(e))


@app.websocket("/ws/chat")
async def websocket_chat(websocket: WebSocket):
    await websocket.accept()
    conversation: List[dict] = []

    try:
        while True:
            data = await websocket.receive_text()
            try:
                msg = json.loads(data)
                user_text = msg.get("content", "").strip()
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

            await websocket.send_text(
                json.dumps({"reply": reply, "done": is_done(reply)})
            )

    except WebSocketDisconnect:
        pass
