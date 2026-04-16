import { useEffect, useRef, useState } from "react";
import { Bot, CheckCircle2, ImageIcon, Loader2, LogOut, Send, Sparkles } from "lucide-react";
import { AccountAvatar } from "../components/AccountAvatar";
import { useAuth } from "../context/useAuth";

const CHAT_BASE_URL = "/api";

function parseJsonObject(text) {
  if (!text || typeof text !== "string") return null;
  const trimmed = text.trim();
  if (!trimmed.startsWith("{") || !trimmed.endsWith("}")) return null;
  try {
    return JSON.parse(trimmed);
  } catch {
    return null;
  }
}

function matchImages(match) {
  const candidates = [
    ...(Array.isArray(match?.images) ? match.images : []),
    ...(Array.isArray(match?.image_urls) ? match.image_urls : []),
    ...(Array.isArray(match?.photo_urls) ? match.photo_urls : []),
    ...(Array.isArray(match?.photos) ? match.photos : []),
  ].filter(Boolean);
  return [...new Set(candidates)].slice(0, 4);
}

function formatBackendResult(action, data) {
  if (!data) return "Done.";

  switch (action) {
    case "create_lost_report": {
      const report = data.report || data;
      return `✅ Your lost item report has been filed!\n\nReport ID: ${report.id || "—"}\nItem: ${report.item_name || "—"}\nStatus: ${report.status || "open"}`;
    }

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
  const seenNotificationIdsRef = useRef(new Set());

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  useEffect(() => {
    if (!user?.id || user?.role !== "passenger") return;

    let cancelled = false;
    async function poll() {
      try {
        const res = await fetch(
          `/passenger/notifications?unread_only=1&limit=20`,
          { credentials: "include" },
        );
        const payload = await res.json().catch(() => null);
        if (!res.ok) return;
        const notes = payload?.notifications || [];
        if (!Array.isArray(notes) || notes.length === 0) return;

        const fresh = notes.filter((n) => {
          const id = String(n?.id || "").trim();
          if (!id) return false;
          return !seenNotificationIdsRef.current.has(id);
        });
        if (fresh.length === 0) return;

        fresh.forEach((n) => {
          const id = String(n?.id || "").trim();
          if (id) seenNotificationIdsRef.current.add(id);
        });

        const byReport = fresh.reduce((acc, n) => {
          const key = String(n?.lost_report_id || "").trim() || "unknown";
          acc[key] = acc[key] || [];
          acc[key].push(n);
          return acc;
        }, {});

        if (cancelled) return;

        const newMessages = Object.entries(byReport).map(
          ([lostReportId, group]) => {
            const matches = group.map((n) => ({
              found_item_id: n.found_item_id,
              item_name: n.item_name,
              similarity_score: n.similarity_score,
              image_urls: n.image_urls,
              primary_image_url: n.primary_image_url,
            }));
            return {
              id: `n-${Date.now()}-${lostReportId}`,
              role: "assistant",
              content:
                "New potential matches were found for your lost item. Select the closest one to file a claim.",
              matchCards: {
                matches,
                lostReportId: lostReportId === "unknown" ? "" : lostReportId,
                claimingId: "",
                claimedId: "",
              },
            };
          },
        );

        setMessages((prev) => [...prev, ...newMessages]);

        const ids = fresh.map((n) => n.id).filter(Boolean);
        await fetch(`/passenger/notifications/read`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({ notification_ids: ids }),
        }).catch(() => null);
      } catch {
        // ignore polling failures (offline, gateway down, etc.)
      }
    }

    poll();
    const t = setInterval(poll, 30_000);
    return () => {
      cancelled = true;
      clearInterval(t);
    };
  }, [user?.id, user?.role]);

  async function requestAssistantReply(conversationMessages) {
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
    return response.json();
  }

  function buildConversationWithNextUserMessage(nextUserText) {
    return messages
      .filter((msg) => msg.id !== "welcome")
      .map((msg) => ({ role: msg.role, content: msg.content }))
      .concat([{ role: "user", content: nextUserText }]);
  }

  async function sendChatText(text) {
    const cleanText = text.trim();
    if (!cleanText || sending) return;

    const userId = `u-${Date.now()}`;
    setMessages((m) => [...m, { id: userId, role: "user", content: cleanText }]);
    setDraft("");
    setSending(true);

    try {
      const data = await requestAssistantReply(
        buildConversationWithNextUserMessage(cleanText),
      );
      const actionPayload = parseJsonObject(data.reply);

      let assistantMessage = {
        id: `a-${Date.now()}`,
        role: "assistant",
        content: data.reply,
      };

      if (data.action && data.action !== "none") {
        if (data.grpc_ok) {
          if (data.action === "search_found_item_matches") {
            const foundMatches = data.grpc_data?.matches || [];
            const lostReportId = actionPayload?.data?.lost_report_id || "";
            assistantMessage = {
              ...assistantMessage,
              content:
                foundMatches.length > 0
                  ? "We found potential matches. Select the closest one and file your claim."
                  : "No matching found items yet. We'll keep looking!",
              matchCards: {
                matches: foundMatches,
                lostReportId,
                claimingId: "",
                claimedId: "",
              },
            };
          } else if (data.action === "create_lost_report") {
            const createdReport = data.grpc_data?.report || null;
            const foundMatches = data.grpc_data?.matches || [];
            if (createdReport?.id && Array.isArray(foundMatches)) {
              assistantMessage = {
                ...assistantMessage,
                content:
                  foundMatches.length > 0
                    ? "✅ Report filed. We also found potential matches — select the closest one to file a claim."
                    : assistantMessage.content,
                matchCards: {
                  matches: foundMatches,
                  lostReportId: createdReport.id,
                  claimingId: "",
                  claimedId: "",
                },
              };
            } else {
              assistantMessage.content = formatBackendResult(
                data.action,
                data.grpc_data,
              );
            }
          } else {
            assistantMessage.content = formatBackendResult(data.action, data.grpc_data);
          }
        } else {
          assistantMessage.content += `\n\n⚠️ Backend error: ${data.grpc_error || "Unknown error"}`;
        }
      }

      setMessages((m) => [...m, assistantMessage]);
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
  }

  function handleSend(e) {
    e.preventDefault();
    void sendChatText(draft);
  }

  async function handleClaimFromMatch(messageId, match) {
    const hostMessage = messages.find((m) => m.id === messageId);
    const lostReportId = hostMessage?.matchCards?.lostReportId;
    if (!lostReportId || !match?.found_item_id) return;

    setMessages((prev) =>
      prev.map((m) =>
        m.id === messageId
          ? {
              ...m,
              matchCards: {
                ...m.matchCards,
                claimingId: match.found_item_id,
              },
            }
          : m,
      ),
    );

    const claimPrompt = `File a claim for my selected match. Respond with JSON only using action "file_claim". Use found_item_id "${match.found_item_id}", lost_report_id "${lostReportId}", and set a concise message saying this is my selected item.`;

    try {
      const data = await requestAssistantReply(
        buildConversationWithNextUserMessage(claimPrompt),
      );

      let replyContent = data.reply;
      if (data.action && data.action !== "none") {
        if (data.grpc_ok) {
          replyContent = formatBackendResult(data.action, data.grpc_data);
        } else {
          replyContent += `\n\n⚠️ Backend error: ${data.grpc_error || "Unknown error"}`;
        }
      }

      setMessages((prev) => [
        ...prev.map((m) =>
          m.id === messageId
            ? {
                ...m,
                matchCards: {
                  ...m.matchCards,
                  claimingId: "",
                  claimedId:
                    data.action === "file_claim" && data.grpc_ok
                      ? match.found_item_id
                      : m.matchCards?.claimedId || "",
                },
              }
            : m,
        ),
        { id: `a-${Date.now()}`, role: "assistant", content: replyContent },
      ]);
    } catch (error) {
      setMessages((prev) =>
        prev.map((m) =>
          m.id === messageId
            ? {
                ...m,
                matchCards: {
                  ...m.matchCards,
                  claimingId: "",
                },
              }
            : m,
        ),
      );
      setMessages((m) => [
        ...m,
        {
          id: `a-${Date.now()}`,
          role: "assistant",
          content: `Unable to file claim right now: ${error.message}`,
        },
      ]);
    }
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
                {msg.role === "assistant" && msg.matchCards?.matches?.length > 0 ? (
                  <div className="mt-3 space-y-2.5">
                    {msg.matchCards.matches.map((match) => {
                      const images = matchImages(match);
                      const isClaiming =
                        msg.matchCards.claimingId === match.found_item_id;
                      const isClaimed =
                        msg.matchCards.claimedId === match.found_item_id;
                      return (
                        <button
                          key={match.found_item_id}
                          type="button"
                          disabled={isClaiming || isClaimed}
                          onClick={() => handleClaimFromMatch(msg.id, match)}
                          className="w-full rounded-xl border border-border/80 bg-background p-3 text-left transition hover:bg-muted/40 disabled:cursor-not-allowed disabled:opacity-75"
                        >
                          <div className="grid grid-cols-4 gap-2">
                            {images.length > 0 ? (
                              images.map((url, idx) => (
                                <img
                                  key={`${match.found_item_id}-${idx}`}
                                  src={url}
                                  alt={`${match.item_name || "Matched item"} ${idx + 1}`}
                                  className="h-16 w-full rounded-md object-cover"
                                />
                              ))
                            ) : (
                              <div className="col-span-4 flex h-16 items-center justify-center rounded-md border border-dashed border-border text-muted-foreground">
                                <ImageIcon className="mr-1.5 h-4 w-4" />
                                No photos available
                              </div>
                            )}
                          </div>
                          <div className="mt-2 space-y-1">
                            <p className="text-sm font-semibold text-foreground">
                              {match.item_name || "Unnamed item"}
                            </p>
                            <p className="text-xs text-muted-foreground">
                              {[match.color, match.brand, match.model]
                                .filter(Boolean)
                                .join(" • ") || "No extra details"}
                            </p>
                            {match.item_description ? (
                              <p className="line-clamp-2 text-xs text-muted-foreground/90">
                                {match.item_description}
                              </p>
                            ) : null}
                          </div>
                          <div className="mt-2 flex items-center justify-between">
                            <span className="text-[11px] text-muted-foreground">
                              Score {(match.similarity_score ?? 0).toFixed(2)}
                            </span>
                            <span className="inline-flex items-center rounded-full border border-border px-2 py-0.5 text-[11px] font-medium">
                              {isClaimed ? (
                                <>
                                  <CheckCircle2 className="mr-1 h-3.5 w-3.5 text-emerald-500" />
                                  Claim filed
                                </>
                              ) : isClaiming ? (
                                <>
                                  <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />
                                  Filing claim...
                                </>
                              ) : (
                                "Select & file claim"
                              )}
                            </span>
                          </div>
                        </button>
                      );
                    })}
                  </div>
                ) : null}
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
