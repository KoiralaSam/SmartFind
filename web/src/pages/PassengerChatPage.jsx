import { useEffect, useRef, useState, useCallback } from "react";
import { Link } from "react-router-dom";
import { Bot, LogOut, Send, Sparkles, UserRound } from "lucide-react";
import { useAuth } from "../context/useAuth";

const CHAT_WS_URL =
  (import.meta.env.VITE_CHAT_API_URL || "http://localhost:8000")
    .replace(/^http/, "ws") + "/ws/chat";

const WELCOME_MESSAGE = {
  id: "welcome",
  role: "assistant",
  content:
    "Hello! I'm so sorry to hear you've lost something — I know how stressful that can be. I'm here to help you file a lost item report. Could you start by telling me what item you lost?",
};

export default function PassengerChatPage() {
  const { user, logout } = useAuth();
  const [messages, setMessages] = useState([WELCOME_MESSAGE]);
  const [draft, setDraft] = useState("");
  const [sending, setSending] = useState(false);
  const [reportSubmitted, setReportSubmitted] = useState(false);
  const [wsError, setWsError] = useState(false);
  const bottomRef = useRef(null);
  const wsRef = useRef(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const connectWs = useCallback(() => {
    const ws = new WebSocket(CHAT_WS_URL);

    ws.onopen = () => {
      setWsError(false);
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        setMessages((m) => [
          ...m,
          { id: `a-${Date.now()}`, role: "assistant", content: data.reply },
        ]);
        if (data.done) {
          setReportSubmitted(true);
        }
      } catch {
        setMessages((m) => [
          ...m,
          { id: `a-${Date.now()}`, role: "assistant", content: event.data },
        ]);
      }
      setSending(false);
    };

    ws.onerror = () => {
      setWsError(true);
      setSending(false);
    };

    ws.onclose = () => {
      wsRef.current = null;
    };

    wsRef.current = ws;
    return ws;
  }, []);

  useEffect(() => {
    const ws = connectWs();
    return () => {
      ws.close();
    };
  }, [connectWs]);

  function handleSend(e) {
    e.preventDefault();
    const text = draft.trim();
    if (!text || sending) return;

    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      connectWs();
      setTimeout(() => sendMessage(text), 300);
      return;
    }

    sendMessage(text);
  }

  function sendMessage(text) {
    setMessages((m) => [
      ...m,
      { id: `u-${Date.now()}`, role: "user", content: text },
    ]);
    setDraft("");
    setSending(true);
    wsRef.current.send(JSON.stringify({ content: text }));
  }

  return (
    <div className="flex min-h-screen flex-col bg-gradient-to-b from-muted/40 via-background to-background">
      {/* Header */}
      <header className="sticky top-0 z-10 border-b border-border/80 bg-background/90 backdrop-blur-md">
        <div className="mx-auto flex h-14 max-w-4xl items-center gap-3 px-3 sm:px-4">
          <div className="flex min-w-0 flex-1 items-center gap-2.5">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-violet-600 to-indigo-600 text-white shadow-sm">
              <Sparkles className="h-4 w-4" aria-hidden />
            </div>
            <div className="min-w-0">
              <h1 className="truncate text-sm font-semibold leading-tight sm:text-base">
                Lost & Found Assistant
              </h1>
              <p className="truncate text-xs text-muted-foreground">
                {wsError ? "Connection error" : user?.name ?? "Connected"}
              </p>
            </div>
          </div>
          {user?.picture && (
            <img
              src={user.picture}
              alt={user.name}
              className="h-8 w-8 shrink-0 rounded-full object-cover"
            />
          )}
          <Link
            to="/"
            onClick={logout}
            className="inline-flex shrink-0 items-center gap-2 rounded-lg border border-border px-3 py-1.5 text-sm font-medium text-foreground shadow-sm transition hover:bg-muted"
          >
            <LogOut className="h-4 w-4" aria-hidden />
            End session
          </Link>
        </div>
      </header>

      <div className="mx-auto flex w-full max-w-4xl flex-1 flex-col overflow-hidden px-3 py-4 sm:px-4">
        {wsError && (
          <div className="mb-3 rounded-lg border border-red-200 bg-red-50 px-4 py-2 text-xs text-red-700">
            Could not connect to the chat service. Make sure the backend is running on{" "}
            <code className="font-mono">{CHAT_WS_URL}</code>.
          </div>
        )}

        {/* Messages */}
        <div className="flex-1 space-y-4 overflow-y-auto pb-4 pr-1">
          {messages.map((msg) => (
            <div
              key={msg.id}
              className={`flex gap-3 ${msg.role === "user" ? "flex-row-reverse" : "flex-row"}`}
            >
              {/* Avatar */}
              <div
                className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-full overflow-hidden sm:h-9 sm:w-9 ${
                  msg.role === "user"
                    ? "bg-foreground text-background"
                    : "bg-gradient-to-br from-violet-500 to-indigo-600 text-white"
                }`}
              >
                {msg.role === "user" ? (
                  user?.picture ? (
                    <img src={user.picture} alt="" className="h-full w-full object-cover" />
                  ) : (
                    <UserRound className="h-4 w-4" aria-hidden />
                  )
                ) : (
                  <Bot className="h-4 w-4" aria-hidden />
                )}
              </div>

              {/* Bubble */}
              <div
                className={`max-w-[min(100%,28rem)] rounded-2xl px-4 py-3 text-sm leading-relaxed shadow-sm sm:text-[15px] ${
                  msg.role === "user"
                    ? "rounded-tr-md border border-border bg-white text-gray-600"
                    : "rounded-tl-md border border-border bg-white text-gray-600"
                }`}
              >
                <p className="whitespace-pre-wrap break-words">{msg.content}</p>
              </div>
            </div>
          ))}

          {/* Typing indicator */}
          {sending && (
            <div className="flex gap-3">
              <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-violet-500 to-indigo-600 text-white sm:h-9 sm:w-9">
                <Bot className="h-4 w-4 animate-pulse" aria-hidden />
              </div>
              <div className="rounded-2xl rounded-tl-md border border-border bg-white px-4 py-3 text-sm text-gray-400">
                <span className="animate-pulse">···</span>
              </div>
            </div>
          )}
          <div ref={bottomRef} />
        </div>

        {/* Footer */}
        {reportSubmitted ? (
          <div className="border-t border-border/80 bg-background/95 pt-4 pb-2 text-center">
            <p className="text-sm font-medium text-green-600">
              Your report has been submitted. We'll be in touch if your item is found.
            </p>
            <Link
              to="/"
              onClick={logout}
              className="mt-3 inline-block text-sm text-muted-foreground underline underline-offset-4 hover:text-foreground"
            >
              Return to home
            </Link>
          </div>
        ) : (
          <form
            onSubmit={handleSend}
            className="border-t border-border/80 bg-background/95 pt-3 backdrop-blur-sm"
          >
            <div className="flex gap-2 rounded-2xl border border-border bg-muted/30 p-1.5 shadow-inner focus-within:ring-2 focus-within:ring-violet-400/30">
              <label htmlFor="chat-input" className="sr-only">Message</label>
              <input
                id="chat-input"
                type="text"
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                placeholder="Type your message…"
                autoComplete="off"
                disabled={wsError}
                className="min-h-11 flex-1 rounded-xl border-0 bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:ring-0 disabled:opacity-50"
              />
              <button
                type="submit"
                disabled={!draft.trim() || sending || wsError}
                className="inline-flex h-11 shrink-0 items-center justify-center gap-2 rounded-xl bg-foreground px-4 text-sm font-medium text-background shadow-sm transition hover:opacity-90 disabled:pointer-events-none disabled:opacity-40"
              >
                <Send className="h-4 w-4" aria-hidden />
                <span className="hidden sm:inline">Send</span>
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}
