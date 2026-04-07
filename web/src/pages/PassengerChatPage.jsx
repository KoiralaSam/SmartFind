import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { Bot, LogOut, Send, Sparkles, UserRound } from "lucide-react";

function buildMockReply(userText) {
  const trimmed = userText.trim();
  if (!trimmed) {
    return "Send a message to get started — I’ll respond here once AI is connected.";
  }
  return `Thanks for your message. Full AI integration is coming soon.\n\nYou wrote: “${trimmed.slice(0, 200)}${trimmed.length > 200 ? "…" : ""}”\n\nFor lost items or routes, you’ll be able to get real answers here shortly.`;
}

export default function PassengerChatPage() {
  const [messages, setMessages] = useState(() => [
    {
      id: "welcome",
      role: "assistant",
      content:
        "Hi — I’m your SmartFind transit assistant (preview). Ask about lost items, routes, or anything else. Real AI replies will be wired up later.",
    },
  ]);
  const [draft, setDraft] = useState("");
  const [sending, setSending] = useState(false);
  const bottomRef = useRef(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  function handleSend(e) {
    e.preventDefault();
    const text = draft.trim();
    if (!text || sending) return;

    const userId = `u-${Date.now()}`;
    setMessages((m) => [
      ...m,
      { id: userId, role: "user", content: text },
    ]);
    setDraft("");
    setSending(true);

    setTimeout(() => {
      setMessages((m) => [
        ...m,
        {
          id: `a-${Date.now()}`,
          role: "assistant",
          content: buildMockReply(text),
        },
      ]);
      setSending(false);
    }, 450);
  }

  return (
    <div className="flex min-h-screen flex-col bg-gradient-to-b from-muted/40 via-background to-background">
      <header className="sticky top-0 z-10 border-b border-border/80 bg-background/90 backdrop-blur-md">
        <div className="mx-auto flex h-14 max-w-4xl items-center gap-3 px-3 sm:px-4">
          <Link
            to="/"
            className="inline-flex shrink-0 items-center gap-2 rounded-lg px-2 py-1.5 text-sm font-medium text-muted-foreground transition hover:bg-muted hover:text-foreground"
          >
            <LogOut className="h-4 w-4" aria-hidden />
            End session
          </Link>
          <div className="flex min-w-0 flex-1 items-center gap-2.5">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-violet-600 to-indigo-600 text-white shadow-sm">
              <Sparkles className="h-4 w-4" aria-hidden />
            </div>
            <div className="min-w-0">
              <h1 className="truncate text-sm font-semibold leading-tight sm:text-base">
                Transit assistant
              </h1>
              <p className="truncate text-xs text-muted-foreground">
                Preview · AI backend not connected yet
              </p>
            </div>
          </div>
        </div>
      </header>

      <div className="mx-auto flex w-full max-w-4xl flex-1 flex-col overflow-hidden px-3 py-4 sm:px-4">
        <div className="flex-1 space-y-4 overflow-y-auto pb-4 pr-1">
          {messages.map((msg) => (
            <div
              key={msg.id}
              className={`flex gap-3 ${msg.role === "user" ? "flex-row-reverse" : "flex-row"}`}
            >
              <div
                className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-full sm:h-9 sm:w-9 ${
                  msg.role === "user"
                    ? "bg-foreground text-background"
                    : "bg-gradient-to-br from-violet-500 to-indigo-600 text-white"
                }`}
              >
                {msg.role === "user" ? (
                  <UserRound className="h-4 w-4" aria-hidden />
                ) : (
                  <Bot className="h-4 w-4" aria-hidden />
                )}
              </div>
              <div
                className={`max-w-[min(100%,28rem)] rounded-2xl px-4 py-3 text-sm leading-relaxed shadow-sm sm:text-[15px] ${
                  msg.role === "user"
                    ? "rounded-tr-md bg-foreground text-background"
                    : "rounded-tl-md border border-border/80 bg-card text-card-foreground"
                }`}
              >
                <p className="whitespace-pre-wrap break-words">{msg.content}</p>
              </div>
            </div>
          ))}
          {sending ? (
            <div className="flex gap-3">
              <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-violet-500 to-indigo-600 text-white sm:h-9 sm:w-9">
                <Bot className="h-4 w-4 animate-pulse" aria-hidden />
              </div>
              <div className="rounded-2xl rounded-tl-md border border-border/80 bg-muted/60 px-4 py-3 text-sm text-muted-foreground">
                …
              </div>
            </div>
          ) : null}
          <div ref={bottomRef} />
        </div>

        <form
          onSubmit={handleSend}
          className="border-t border-border/80 bg-background/95 pt-3 backdrop-blur-sm"
        >
          <div className="flex gap-2 rounded-2xl border border-border bg-muted/30 p-1.5 shadow-inner focus-within:ring-2 focus-within:ring-ring/30">
            <label htmlFor="chat-input" className="sr-only">
              Message
            </label>
            <input
              id="chat-input"
              type="text"
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              placeholder="Message the assistant…"
              autoComplete="off"
              className="min-h-11 flex-1 rounded-xl border-0 bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:ring-0"
            />
            <button
              type="submit"
              disabled={!draft.trim() || sending}
              className="inline-flex h-11 shrink-0 items-center justify-center gap-2 rounded-xl bg-foreground px-4 text-sm font-medium text-background transition hover:opacity-90 disabled:pointer-events-none disabled:opacity-40"
            >
              <Send className="h-4 w-4" aria-hidden />
              <span className="hidden sm:inline">Send</span>
            </button>
          </div>
          <p className="mt-2 text-center text-[11px] text-muted-foreground">
            Google sign-in and live AI will be enabled in a future update.
          </p>
        </form>
      </div>
    </div>
  );
}
