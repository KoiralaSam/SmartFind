import { useEffect, useRef, useState } from "react";
import { Bot, CheckCircle2, ImageIcon, Loader2, Mic, MicOff, Send, Sparkles } from "lucide-react";
import { AccountAvatar } from "../components/AccountAvatar";
import { NotificationsPanel } from "../components/NotificationsPanel";
import { passengerFileClaim } from "../api/gateway";
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
    match?.primary_image_url,
    match?.primaryImageUrl,
    match?.primary_image,
    match?.image,
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
          `${i + 1}. ${r.item_name || "Unnamed"} - ${r.status || "open"} (ID: ${r.id})`,
      );
      return `Here are your lost item reports:\n\n${lines.join("\n")}`;
    }
    case "delete_lost_report":
      return "The lost item report has been deleted.";
    case "search_found_item_matches": {
      const matches = data.matches || [];
      if (matches.length === 0)
        return "No matching found items yet. We will keep looking.";
      const lines = matches.map(
        (m, i) =>
          `${i + 1}. ${m.item_name || "Unnamed"} - ${m.color || ""} ${m.brand || ""} (score: ${(m.similarity_score ?? 0).toFixed(2)}, ID: ${m.found_item_id})`,
      );
      return `We found potential matches:\n\n${lines.join("\n")}`;
    }
    case "file_claim":
      return `✅ Your claim has been filed!\n\nClaim ID: ${data.id || "—"}\nStatus: ${data.status || "pending"}`;

    case "list_my_claims": {
      const claims = data.claims || [];
      if (claims.length === 0) return "You have no claims on file.";
      const lines = claims.map(
        (c, i) =>
          `${i + 1}. ${c.status || "pending"} — Claim ${c.id} (Found item: ${c.item_id}${c.lost_report_id ? `, Lost report: ${c.lost_report_id}` : ""})`,
      );
      return `Here are your claims:\n\n${lines.join("\n")}`;
    }
    default:
      return "Done.";
  }
}

export default function PassengerChatPage() {
  const { user } = useAuth();
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
  const [recording, setRecording] = useState(false);
  const bottomRef = useRef(null);
  const messagesRef = useRef([]);
  const seenNotificationIdsRef = useRef(new Set());
  const pendingNotificationIdsRef = useRef(new Set());
  const mediaRecorderRef = useRef(null);
  const voiceSocketRef = useRef(null);
  const lastInterimRef = useRef("");
  const voiceEnabledRef = useRef(false);
  const activeAudioRef = useRef(null);
  const sendingRef = useRef(false);
  const pendingVoiceFinalsRef = useRef([]);
  const lastSubmittedVoiceRef = useRef({ text: "", ts: 0 });
  const ttsPlaybackTokenRef = useRef(0);

  useEffect(() => {
    messagesRef.current = messages;
  }, [messages]);

  useEffect(() => {
    seenNotificationIdsRef.current.clear();
    pendingNotificationIdsRef.current.clear();
    voiceEnabledRef.current = false;
  }, [user?.id]);

  useEffect(() => {
    sendingRef.current = sending;
  }, [sending]);

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
          return (
            !seenNotificationIdsRef.current.has(id) &&
            !pendingNotificationIdsRef.current.has(id)
          );
        });
        const idsToMarkRead = notes
          .map((n) => String(n?.id || "").trim())
          .filter(Boolean)
          .filter((id) => pendingNotificationIdsRef.current.has(id));

        const freshIds = fresh
          .map((n) => String(n?.id || "").trim())
          .filter(Boolean);
        const markIds = [...new Set([...idsToMarkRead, ...freshIds])];
        if (markIds.length === 0) return;

        const byReport = fresh.reduce((acc, n) => {
          const key = String(n?.lost_report_id || "").trim() || "unknown";
          acc[key] = acc[key] || [];
          acc[key].push(n);
          return acc;
        }, {});

        if (cancelled) return;

        // Mark these as pending so we don't re-display duplicates while we retry mark-read.
        markIds.forEach((id) => pendingNotificationIdsRef.current.add(id));

        if (Object.keys(byReport).length > 0) {
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
        }

        const readRes = await fetch(`/passenger/notifications/read`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({ notification_ids: markIds }),
        }).catch(() => null);

        if (readRes?.ok) {
          markIds.forEach((id) => {
            pendingNotificationIdsRef.current.delete(id);
            seenNotificationIdsRef.current.add(id);
          });
        }
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
    const snapshot = Array.isArray(messagesRef.current) ? messagesRef.current : [];
    // Keep context bounded to avoid sending unbounded history.
    const tail = snapshot.slice(-40);
    return tail
      .map((msg) => ({ role: msg.role, content: msg.content }))
      .concat([{ role: "user", content: nextUserText }]);
  }

  async function sendChatText(text) {
    const cleanText = text.trim();
    if (!cleanText || sendingRef.current) return;

    const userId = `u-${Date.now()}`;
    setMessages((m) => [...m, { id: userId, role: "user", content: cleanText }]);
    setDraft("");
    sendingRef.current = true;
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
          } else if (data.action === "check_my_lost_item") {
            const foundMatches = data.grpc_data?.matches || [];
            const lostReportId = data.grpc_data?.lost_report_id || "";
            const hasReport = Boolean(lostReportId);
            assistantMessage = {
              ...assistantMessage,
              content: hasReport
                ? foundMatches.length > 0
                  ? "I checked your most recent report and found potential matches. Select the closest one to file a claim."
                  : "I checked your most recent report — no matching found items yet. We’ll keep looking!"
                : "I couldn’t find an existing open lost report for you. If you lost something recently, tell me what it was and I’ll file a report.",
              ...(hasReport
                ? {
                    matchCards: {
                      matches: foundMatches,
                      lostReportId,
                      claimingId: "",
                      claimedId: "",
                    },
                  }
                : {}),
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
                    : "✅ Report filed. No matching found items yet — we’ll keep looking!",
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
          console.warn("Backend action failed:", data.action, data.grpc_error);
          assistantMessage.content =
            "I couldn’t complete that request right now. Please try again in a moment.";
        }
      }

      setMessages((m) => [...m, assistantMessage]);

      // If voice mode is enabled, speak the assistant reply via Deepgram TTS.
      if (voiceEnabledRef.current && assistantMessage?.content) {
        const messageId = assistantMessage.id;
        try {
          const ttsRes = await fetch(`${CHAT_BASE_URL}/tts`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ text: assistantMessage.content }),
          });
          const ttsPayload = await ttsRes.json().catch(() => null);
          if (ttsRes.ok && ttsPayload?.audio_base64) {
            const mime = ttsPayload?.mime || "audio/mpeg";
            const b64 = String(ttsPayload.audio_base64);
            const bytes = Uint8Array.from(atob(b64), (c) => c.charCodeAt(0));
            const blob = new Blob([bytes], { type: mime });
            const url = URL.createObjectURL(blob);
            const playbackToken = ++ttsPlaybackTokenRef.current;

            const previousAudio = activeAudioRef.current;
            if (previousAudio) {
              previousAudio.onended = null;
              previousAudio.onerror = null;
              try {
                previousAudio.pause();
              } catch {
                // ignore
              }
            }

            setMessages((prev) =>
              prev.map((msg) =>
                msg.id === messageId ? { ...msg, audioUrl: url } : msg,
              ),
            );

            // Pause mic capture while the agent is speaking to reduce feedback loops.
            const recorder = mediaRecorderRef.current;
            if (recording && recorder?.state === "recording" && recorder.pause) {
              try {
                recorder.pause();
              } catch {
                // ignore
              }
            }

            const audio = new Audio(url);
            activeAudioRef.current = audio;
            audio.onended = () => {
              if (ttsPlaybackTokenRef.current !== playbackToken) return;
              activeAudioRef.current = null;
              if (recording && recorder?.state === "paused" && recorder.resume) {
                try {
                  recorder.resume();
                } catch {
                  // ignore
                }
              }
            };
            audio.onerror = () => {
              if (ttsPlaybackTokenRef.current !== playbackToken) return;
              activeAudioRef.current = null;
              if (recording && recorder?.state === "paused" && recorder.resume) {
                try {
                  recorder.resume();
                } catch {
                  // ignore
                }
              }
            };
            try {
              await audio.play();
            } catch {
              // Autoplay may be blocked on some browsers. Audio controls are still shown.
            }
          }
        } catch {
          // ignore TTS failures
        }
      }
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
      sendingRef.current = false;
      setSending(false);
      if (voiceEnabledRef.current) {
        const queuedText = pendingVoiceFinalsRef.current.shift();
        if (queuedText) {
          void sendChatText(queuedText);
        }
      } else {
        pendingVoiceFinalsRef.current = [];
      }
    }
  }

  function handleSend(e) {
    e.preventDefault();
    void sendChatText(draft);
  }

  function wsBase() {
    const proto = window.location.protocol === "https:" ? "wss" : "ws";
    return `${proto}://${window.location.host}`;
  }

  async function startVoice() {
    if (recording) return;
    if (!navigator?.mediaDevices?.getUserMedia) {
      setMessages((m) => [
        ...m,
        {
          id: `a-${Date.now()}`,
          role: "assistant",
          content: "Voice input isn’t supported in this browser. Please type your message instead.",
        },
      ]);
      return;
    }

    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      const socket = new WebSocket(`${wsBase()}${CHAT_BASE_URL}/voice`);
      socket.binaryType = "arraybuffer";
      voiceSocketRef.current = socket;
      lastInterimRef.current = "";

      socket.onmessage = (evt) => {
        if (!voiceEnabledRef.current || voiceSocketRef.current !== socket) return;
        try {
          const msg = JSON.parse(evt.data);
          if (msg?.type === "transcript") {
            const t = String(msg?.text || "");
            lastInterimRef.current = t;
            setDraft(t);
            return;
          }
          if (msg?.type === "final") {
            const t = String(msg?.text || "").trim();
            setDraft("");
            if (t) {
              const now = Date.now();
              const last = lastSubmittedVoiceRef.current;
              const duplicate =
                last.text.toLowerCase() === t.toLowerCase() && now - last.ts < 3000;
              if (!duplicate) {
                lastSubmittedVoiceRef.current = { text: t, ts: now };
                if (sendingRef.current) {
                  pendingVoiceFinalsRef.current.push(t);
                } else {
                  void sendChatText(t);
                }
              }
            }
            return;
          }
          if (msg?.type === "error") {
            console.warn("Voice error:", msg?.reason || "unknown");
            setRecording(false);
            voiceEnabledRef.current = false;
            setDraft("");
            setMessages((m) => [
              ...m,
              {
                id: `a-${Date.now()}`,
                role: "assistant",
                content:
                  "I couldn’t use voice input right now. Please try again or type your message.",
              },
            ]);
          }
        } catch {
          // ignore
        }
      };

      socket.onopen = () => {
        const preferredTypes = [
          "audio/webm;codecs=opus",
          "audio/webm",
          "audio/mp4",
        ];
        const mimeType = preferredTypes.find((t) =>
          window.MediaRecorder?.isTypeSupported?.(t),
        );

        const recorder = new MediaRecorder(stream, mimeType ? { mimeType } : undefined);
        mediaRecorderRef.current = recorder;
        recorder.ondataavailable = async (e) => {
          if (!e?.data || e.data.size === 0) return;
          const s = voiceSocketRef.current;
          if (!s || s.readyState !== WebSocket.OPEN) return;
          const buf = await e.data.arrayBuffer();
          s.send(buf);
        };
        recorder.onstop = () => {
          stream.getTracks().forEach((t) => t.stop());
        };
        recorder.start(250);
        setRecording(true);
        voiceEnabledRef.current = true;
      };

      socket.onerror = () => {
        setRecording(false);
        setDraft("");
        try {
          stream.getTracks().forEach((t) => t.stop());
        } catch {
          // ignore
        }
        setMessages((m) => [
          ...m,
          {
            id: `a-${Date.now()}`,
            role: "assistant",
            content:
              "I couldn’t start voice input. Please try again or type your message.",
          },
        ]);
      };
    } catch (e) {
      setMessages((m) => [
        ...m,
        {
          id: `a-${Date.now()}`,
          role: "assistant",
          content:
            "Could not access your microphone. Please allow microphone permission or use typing.",
        },
      ]);
    }
  }

  function stopVoice() {
    setRecording(false);
    voiceEnabledRef.current = false;
    setDraft("");

    pendingVoiceFinalsRef.current = [];
    lastSubmittedVoiceRef.current = { text: "", ts: 0 };
    lastInterimRef.current = "";

    ttsPlaybackTokenRef.current += 1;
    const activeAudio = activeAudioRef.current;
    if (activeAudio) {
      activeAudio.onended = null;
      activeAudio.onerror = null;
      try {
        activeAudio.pause();
      } catch {
        // ignore
      }
    }
    activeAudioRef.current = null;

    const socket = voiceSocketRef.current;
    if (socket) {
      try {
        socket.onmessage = null;
        socket.onopen = null;
        socket.onerror = null;
        socket.onclose = null;
      } catch {
        // ignore
      }
      try {
        if (socket.readyState === WebSocket.OPEN) {
          socket.send("__STOP__");
        }
      } catch {
        // ignore
      }
      try {
        socket.close();
      } catch {
        // ignore
      }
    }
    voiceSocketRef.current = null;

    const recorder = mediaRecorderRef.current;
    if (recorder) {
      try {
        recorder.ondataavailable = null;
      } catch {
        // ignore
      }
      try {
        if (recorder.state !== "inactive") {
          recorder.stop();
        }
      } catch {
        // ignore
      }
      try {
        recorder.stream?.getTracks?.().forEach((t) => {
          try {
            t.stop();
          } catch {
            // ignore
          }
        });
      } catch {
        // ignore
      }
    }
    mediaRecorderRef.current = null;
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

    try {
      // Canonical path: the api-gateway's POST /passenger/claims wraps the
      // passenger-service FileClaim gRPC and resolves the passenger from the
      // forwarded session token. The notifications drawer uses the same helper,
      // so auth, validation, and duplicate-claim errors go through one funnel.
      const claim = await passengerFileClaim({
        foundItemId: match.found_item_id,
        lostReportId,
        message: "I believe this is my item.",
      });

      setMessages((prev) => [
        ...prev.map((m) =>
          m.id === messageId
            ? {
                ...m,
                matchCards: {
                  ...m.matchCards,
                  claimingId: "",
                  claimedId: match.found_item_id,
                },
              }
            : m,
        ),
        {
          id: `a-${Date.now()}`,
          role: "assistant",
          content: formatBackendResult("file_claim", claim || {}),
        },
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
    <div className="h-full overflow-hidden">
      <div className="flex h-full w-full flex-col px-1 py-2 sm:px-0 sm:py-3">
        <header className="shrink-0 border-b border-border/70 px-2 py-2 sm:px-0">
          <div className="flex items-center justify-between gap-3">
            <div className="flex min-w-0 items-center gap-2.5 sm:gap-3">
              <div className="flex h-8 w-8 shrink-0 items-center justify-center text-foreground sm:h-9 sm:w-9">
                <Sparkles className="h-4 w-4" aria-hidden />
              </div>
              <div className="min-w-0">
                <h1 className="truncate text-sm font-semibold tracking-tight sm:text-base">
                  SmartFind Passenger Assistant
                </h1>
                <p className="truncate text-[11px] text-muted-foreground sm:text-sm">
                  Report, track, and claim lost items in one place
                </p>
              </div>
            </div>
            <div className="shrink-0">
              <NotificationsPanel />
            </div>
          </div>
        </header>

        <div className="mx-auto flex w-full max-w-4xl flex-1 flex-col overflow-hidden px-2 sm:px-4">
          <div className="flex-1 space-y-3 overflow-y-auto py-3 pr-1 sm:space-y-4 sm:py-4">
            {messages.map((msg) => (
              <div
                key={msg.id}
                className={`flex gap-2.5 sm:gap-3 ${msg.role === "user" ? "flex-row-reverse" : "flex-row"}`}
              >
                {msg.role === "user" ? (
                  <AccountAvatar user={user} sizeClass="h-8 w-8 sm:h-9 sm:w-9" />
                ) : (
                  <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-violet-500 to-indigo-600 text-white sm:h-9 sm:w-9">
                    <Bot className="h-4 w-4" aria-hidden />
                  </div>
                )}
                <div
                  className={`max-w-[min(100%,24rem)] sm:max-w-[min(100%,28rem)] rounded-2xl px-3 py-2.5 text-sm leading-relaxed shadow-sm sm:px-4 sm:py-3 sm:text-[15px] ${
                    msg.role === "user"
                      ? "rounded-tr-md border border-border/80 bg-card text-card-foreground"
                      : "rounded-tl-md border border-border/80 bg-card text-card-foreground"
                  }`}
                >
                  <p className="whitespace-pre-wrap break-words">{msg.content}</p>
                  {msg.role === "assistant" && msg.audioUrl ? (
                    <div className="mt-3">
                      <audio
                        src={msg.audioUrl}
                        controls
                        autoPlay={recording}
                        className="w-full"
                      />
                    </div>
                  ) : null}
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
                          <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
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
                  Thinking...
                </div>
              </div>
            ) : null}
            <div ref={bottomRef} />
          </div>

          <form
            onSubmit={handleSend}
            className="shrink-0 border-t border-border/70 bg-background py-2.5 sm:py-3"
          >
            <div className="flex gap-2 rounded-2xl border border-border/70 bg-card/95 p-1.5 transition focus-within:border-border focus-within:ring-2 focus-within:ring-ring/20">
              <label htmlFor="chat-input" className="sr-only">
                Message
              </label>
              <input
                id="chat-input"
                type="text"
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                placeholder={recording ? "Listening..." : "Message the assistant..."}
                autoComplete="off"
                className="min-h-11 flex-1 rounded-xl border-0 bg-transparent px-2.5 text-sm text-foreground outline-none placeholder:text-muted-foreground focus:ring-0 sm:px-3"
              />
              <button
                type="button"
                onClick={() => (recording ? stopVoice() : startVoice())}
                disabled={sending}
                className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-border/70 bg-background text-foreground transition hover:bg-muted/30 disabled:pointer-events-none disabled:opacity-40"
                title={recording ? "Stop voice" : "Start voice"}
              >
                {recording ? (
                  <MicOff className="h-4 w-4" aria-hidden />
                ) : (
                  <Mic className="h-4 w-4" aria-hidden />
                )}
              </button>
              <button
                type="submit"
                disabled={!draft.trim() || sending}
                className="inline-flex h-11 shrink-0 items-center justify-center gap-2 rounded-xl border border-border/70 bg-foreground px-4 text-sm font-medium text-background transition hover:opacity-95 disabled:pointer-events-none disabled:opacity-40"
              >
                <Send className="h-4 w-4" aria-hidden />
                <span className="hidden sm:inline">Send</span>
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}
