import { useEffect, useRef, useState } from "react";
import { Bot, LogOut, Send, Sparkles } from "lucide-react";
import { AccountAvatar } from "../components/AccountAvatar";
import { useAuth } from "../context/useAuth";

const CHAT_BASE_URL = "/api";

function formatBackendResult(action, data) {
  if (!data) return "Done.";

  switch (action) {
    case "create_lost_report":
      return `✅ Your lost item report has been filed!\n\nReport ID: ${data.id || "—"}\nItem: ${data.item_name || "—"}\nStatus: ${data.status || "open"}`;

    case "list_lost_reports": {
      const reports = data.reports || [];
      if (reports.length === 0) return "You have no lost item reports on file.";
      const lines = reports.map(
        (r, i) =>
          `${i + 1}. ${r.item_name || "Unnamed"} — ${r.status || "open"} (ID: ${r.id})`,
      );
      return `Here are your lost item reports:\n\n${lines.join("\n")}`;
    }

    case "delete_lost_report":
      return "✅ The lost item report has been deleted.";

    case "search_found_item_matches": {
      const matches = data.matches || [];
      if (matches.length === 0)
        return "No matching found items yet. We'll keep looking!";
      const lines = matches.map(
        (m, i) =>
          `${i + 1}. ${m.item_name || "Unnamed"} — ${m.color || ""} ${m.brand || ""} (score: ${(m.similarity_score ?? 0).toFixed(2)}, ID: ${m.found_item_id})`,
      );
      return `We found potential matches:\n\n${lines.join("\n")}`;
    }

    case "file_claim":
      return `✅ Your claim has been filed!\n\nClaim ID: ${data.id || "—"}\nStatus: ${data.status || "pending"}`;

    default:
      return "Done.";
  }
}

export default function PassengerChatPage() {
  const { user, logout } = useAuth();
  const [messages, setMessages] = useState(() => [
    {
      id: "welcome",
      role: "assistant",
      content:
        "Hello! I'm so sorry to hear you've lost something — I know how stressful that can be. I'm here to help you file a lost item report. Could you start by telling me what item you lost?",
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
    setMessages((m) => [...m, { id: userId, role: "user", content: text }]);
    setDraft("");
    setSending(true);

    (async () => {
      try {
        const conversationMessages = messages
          .filter((msg) => msg.id !== "welcome")
          .map((msg) => ({ role: msg.role, content: msg.content }))
          .concat([{ role: "user", content: text }]);

        const response = await fetch(`${CHAT_BASE_URL}/chat`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            messages: conversationMessages,
            passenger_id: user?.id || null,
            forwarded_token: user?.sessionToken || null,
          }),
        });

        if (!response.ok) {
          const errBody = await response.json().catch(() => null);
          const detail = errBody?.detail || response.statusText;
          throw new Error(detail);
        }

        const data = await response.json();

        let replyContent = data.reply;
        if (data.action && data.action !== "none") {
          if (data.grpc_ok) {
            replyContent = formatBackendResult(data.action, data.grpc_data);
          } else {
            replyContent += `\n\n⚠️ Backend error: ${data.grpc_error || "Unknown error"}`;
          }
        }

        setMessages((m) => [
          ...m,
          { id: `a-${Date.now()}`, role: "assistant", content: replyContent },
        ]);
      } catch (error) {
        console.error("Chat error:", error);
        setMessages((m) => [
          ...m,
          {
            id: `a-${Date.now()}`,
            role: "assistant",
            content: `Something went wrong: ${error.message}`,
          },
        ]);
      } finally {
        setSending(false);
      }
    })();
  }

  return (
    <div className="flex h-screen flex-col overflow-hidden bg-gradient-to-b from-muted/40 via-background to-background">
      <header className="shrink-0 border-b border-border/80 bg-background/90 backdrop-blur-md">
        <div className="mx-auto flex h-14 max-w-4xl items-center justify-between px-3 sm:px-4">
          <div className="flex min-w-0 flex-1 items-center gap-2.5">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-violet-600 to-indigo-600 text-white shadow-sm">
              <Sparkles className="h-4 w-4" aria-hidden />
            </div>
            <div className="min-w-0">
              <h1 className="truncate text-sm font-semibold leading-tight sm:text-base">
                Transit assistant
              </h1>
              <p className="truncate text-xs text-muted-foreground">
                Lost & Found intake assistant
              </p>
            </div>
          </div>
          <div className="flex shrink-0 items-center gap-2 sm:gap-3">
            {user ? (
              <>
                <AccountAvatar user={user} sizeClass="h-8 w-8 sm:h-9 sm:w-9" />
                <span className="hidden max-w-[100px] truncate text-xs text-muted-foreground sm:inline sm:max-w-[160px] sm:text-sm">
                  {user.name}
                </span>
              </>
            ) : null}
            <button
              type="button"
              onClick={() => logout()}
              className="inline-flex shrink-0 items-center gap-1.5 rounded-full border border-border px-3 py-1.5 text-xs font-medium transition hover:bg-muted sm:text-sm"
            >
              <LogOut className="h-3.5 w-3.5" aria-hidden />
              End session
            </button>
          </div>
        </div>
      </header>

      <div className="mx-auto flex w-full max-w-4xl flex-1 flex-col overflow-hidden px-3 sm:px-4">
        <div className="flex-1 space-y-4 overflow-y-auto py-4 pr-1">
          {messages.map((msg) => (
            <div
              key={msg.id}
              className={`flex gap-3 ${msg.role === "user" ? "flex-row-reverse" : "flex-row"}`}
            >
              {msg.role === "user" ? (
                <AccountAvatar
                  user={user}
                  sizeClass="h-8 w-8 sm:h-9 sm:w-9"
                />
              ) : (
                <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-violet-500 to-indigo-600 text-white sm:h-9 sm:w-9">
                  <Bot className="h-4 w-4" aria-hidden />
                </div>
              )}
              <div
                className={`max-w-[min(100%,28rem)] rounded-2xl px-4 py-3 text-sm leading-relaxed shadow-sm sm:text-[15px] ${
                  msg.role === "user"
                    ? "rounded-tr-md border border-border/80 bg-card text-card-foreground"
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
          className="shrink-0 border-t border-border/80 bg-background/95 py-3 backdrop-blur-sm"
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
        </form>
      </div>
    </div>
  );
}
