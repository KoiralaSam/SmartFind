import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Bell, BellDot, CheckCheck, Loader2, ShieldCheck, X } from "lucide-react";
import {
  passengerFileClaim,
  passengerListClaims,
  passengerListNotifications,
  passengerMarkNotificationsRead,
} from "../api/gateway";

const POLL_INTERVAL_MS = 30000;
const UNREAD_BADGE_CAP = 9;

function formatRelativeTime(iso) {
  if (!iso) return "";
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return "";
  const diff = Date.now() - t;
  if (diff < 0) return "just now";
  const s = Math.floor(diff / 1000);
  if (s < 45) return "just now";
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  const d = Math.floor(h / 24);
  if (d < 7) return `${d}d ago`;
  return new Date(iso).toLocaleDateString();
}

function scorePercent(score) {
  const n = Number(score);
  if (!Number.isFinite(n)) return "";
  return `${Math.round(Math.max(0, Math.min(1, n)) * 100)}% match`;
}

function isUnread(n) {
  if (!n) return false;
  const raw = n.read_at;
  if (!raw) return true;
  // Protobuf-encoded zero timestamp comes back as 0001-01-01T00:00:00Z.
  const t = new Date(raw).getTime();
  return !Number.isFinite(t) || t <= 0 || String(raw).startsWith("0001-");
}

export function NotificationsPanel() {
  const [open, setOpen] = useState(false);
  const [notifications, setNotifications] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [busyId, setBusyId] = useState("");
  const [claimedIds, setClaimedIds] = useState(() => new Set());
  const [claimedItemIds, setClaimedItemIds] = useState(() => new Set());
  const buttonRef = useRef(null);
  const panelRef = useRef(null);

  const unreadCount = useMemo(
    () => notifications.filter(isUnread).length,
    [notifications],
  );

  const refreshClaimedItems = useCallback(async () => {
    try {
      const res = await passengerListClaims({ limit: 200, offset: 0 });
      const next = new Set();
      for (const c of Array.isArray(res?.claims) ? res.claims : []) {
        const st = String(c?.status || "").toLowerCase();
        if (st !== "pending" && st !== "approved") continue;
        const itemId = String(c?.item_id || "").trim();
        if (itemId) next.add(itemId);
      }
      setClaimedItemIds(next);
    } catch {
      // non-blocking; server still prevents duplicates
    }
  }, []);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [res] = await Promise.all([
        passengerListNotifications({ limit: 30 }),
        refreshClaimedItems(),
      ]);
      setNotifications(Array.isArray(res?.notifications) ? res.notifications : []);
    } catch (e) {
      setError(e?.message || "Failed to load notifications.");
    } finally {
      setLoading(false);
    }
  }, [refreshClaimedItems]);

  useEffect(() => {
    let cancelled = false;
    let timer = null;
    async function tick() {
      if (cancelled) return;
      try {
        const [res] = await Promise.all([
          passengerListNotifications({ limit: 30 }),
          refreshClaimedItems(),
        ]);
        if (!cancelled) {
          setNotifications(Array.isArray(res?.notifications) ? res.notifications : []);
        }
      } catch {
        // network blips shouldn't break polling; surface only on manual reload.
      }
    }
    tick();
    timer = setInterval(tick, POLL_INTERVAL_MS);
    return () => {
      cancelled = true;
      if (timer) clearInterval(timer);
    };
  }, [refreshClaimedItems]);

  useEffect(() => {
    if (!open) return undefined;
    function onDocClick(e) {
      if (
        panelRef.current?.contains(e.target) ||
        buttonRef.current?.contains(e.target)
      ) {
        return;
      }
      setOpen(false);
    }
    function onKey(e) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("mousedown", onDocClick);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDocClick);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  async function handleMarkAllRead() {
    const ids = notifications.filter(isUnread).map((n) => n.id).filter(Boolean);
    if (ids.length === 0) return;
    const now = new Date().toISOString();
    setNotifications((prev) =>
      prev.map((n) => (ids.includes(n.id) ? { ...n, read_at: now } : n)),
    );
    try {
      await passengerMarkNotificationsRead(ids);
    } catch {
      // rollback would be disruptive; a refresh on next poll recovers truth.
    }
  }

  async function handleFileClaim(n) {
    if (!n?.id || !n?.found_item_id || !n?.lost_report_id) return;
    setBusyId(n.id);
    setError("");
    try {
      await passengerFileClaim({
        foundItemId: n.found_item_id,
        lostReportId: n.lost_report_id,
        message: "I believe this is my item.",
      });
      setClaimedIds((prev) => {
        const next = new Set(prev);
        next.add(n.id);
        return next;
      });
      const now = new Date().toISOString();
      setNotifications((prev) =>
        prev.map((x) => (x.id === n.id ? { ...x, read_at: now } : x)),
      );
      passengerMarkNotificationsRead([n.id]).catch(() => {});
    } catch (e) {
      const msg = e?.message || "Failed to file claim.";
      if (/already|duplicate/i.test(msg)) {
        setClaimedIds((prev) => {
          const next = new Set(prev);
          next.add(n.id);
          return next;
        });
      } else {
        setError(msg);
      }
    } finally {
      setBusyId("");
    }
  }

  const Badge = () => {
    if (unreadCount <= 0) return null;
    const label = unreadCount > UNREAD_BADGE_CAP ? `${UNREAD_BADGE_CAP}+` : String(unreadCount);
    return (
      <span className="absolute -right-1 -top-1 inline-flex h-4 min-w-[1rem] items-center justify-center rounded-full bg-red-600 px-1 text-[10px] font-semibold leading-none text-white shadow-sm">
        {label}
      </span>
    );
  };

  return (
    <div className="relative">
      <button
        ref={buttonRef}
        type="button"
        onClick={() => {
          setOpen((v) => !v);
          if (!open) {
            load();
          }
        }}
        className="relative inline-flex h-9 w-9 items-center justify-center rounded-lg border border-border/70 text-muted-foreground transition hover:bg-muted/60 hover:text-foreground"
        aria-label="Notifications"
        title="Notifications"
      >
        {unreadCount > 0 ? (
          <BellDot className="h-4 w-4" aria-hidden />
        ) : (
          <Bell className="h-4 w-4" aria-hidden />
        )}
        <Badge />
      </button>

      {open ? (
        <div
          ref={panelRef}
          className="absolute right-0 z-40 mt-2 flex w-[22rem] max-w-[90vw] flex-col rounded-xl border border-border/80 bg-popover shadow-xl"
          role="dialog"
          aria-label="Notifications"
        >
          <div className="flex items-center justify-between gap-2 border-b border-border/70 px-3 py-2">
            <div className="flex items-center gap-2">
              <ShieldCheck className="h-4 w-4 text-muted-foreground" aria-hidden />
              <p className="text-sm font-semibold">Match notifications</p>
            </div>
            <div className="flex items-center gap-1">
              {unreadCount > 0 ? (
                <button
                  type="button"
                  onClick={handleMarkAllRead}
                  className="inline-flex h-7 items-center gap-1 rounded-md px-2 text-xs font-medium text-muted-foreground transition hover:bg-muted/70 hover:text-foreground"
                  title="Mark all as read"
                >
                  <CheckCheck className="h-3.5 w-3.5" aria-hidden />
                  Mark all read
                </button>
              ) : null}
              <button
                type="button"
                onClick={() => setOpen(false)}
                className="inline-flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground transition hover:bg-muted/70 hover:text-foreground"
                aria-label="Close notifications"
              >
                <X className="h-3.5 w-3.5" aria-hidden />
              </button>
            </div>
          </div>

          <div className="max-h-[26rem] overflow-y-auto p-2">
            {loading && notifications.length === 0 ? (
              <div className="flex items-center justify-center py-8 text-sm text-muted-foreground">
                <Loader2 className="mr-2 h-4 w-4 animate-spin" aria-hidden />
                Loading…
              </div>
            ) : error ? (
              <div className="px-3 py-6 text-sm text-red-600">{error}</div>
            ) : notifications.length === 0 ? (
              <div className="px-3 py-8 text-center text-sm text-muted-foreground">
                <p>No match notifications yet.</p>
                <p className="mt-1 text-xs">
                  We'll ping you here (and by email) when a found item looks like yours.
                </p>
              </div>
            ) : (
              <ul className="space-y-2">
                {notifications.map((n) => {
                  const unread = isUnread(n);
                  const claimed =
                    claimedIds.has(n.id) ||
                    claimedItemIds.has(String(n?.found_item_id || "").trim());
                  const busy = busyId === n.id;
                  return (
                    <li
                      key={n.id}
                      className={`rounded-lg border px-3 py-2.5 transition ${
                        unread
                          ? "border-primary/40 bg-primary/5"
                          : "border-border/70 bg-background"
                      }`}
                    >
                      <div className="flex gap-3">
                        {n.primary_image_url ? (
                          <img
                            src={n.primary_image_url}
                            alt={n.item_name || "Match"}
                            className="h-14 w-14 shrink-0 rounded-md border border-border/70 object-cover"
                            loading="lazy"
                          />
                        ) : (
                          <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-md border border-dashed border-border/70 bg-muted/40 text-[10px] text-muted-foreground">
                            No photo
                          </div>
                        )}
                        <div className="min-w-0 flex-1">
                          <div className="flex items-baseline justify-between gap-2">
                            <p className="truncate text-sm font-semibold">
                              {n.item_name || "Potential match"}
                            </p>
                            <span className="shrink-0 text-[11px] text-muted-foreground">
                              {formatRelativeTime(n.created_at)}
                            </span>
                          </div>
                          <p className="mt-0.5 text-[11px] text-muted-foreground">
                            {scorePercent(n.similarity_score) || "Similarity unknown"}
                          </p>
                          <div className="mt-2 flex items-center gap-2">
                            <button
                              type="button"
                              disabled={busy || claimed}
                              onClick={() => handleFileClaim(n)}
                              className="inline-flex h-7 items-center gap-1 rounded-md border border-border/80 bg-background px-2 text-xs font-medium text-foreground transition hover:bg-muted/70 disabled:cursor-not-allowed disabled:opacity-60"
                            >
                              {busy ? (
                                <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden />
                              ) : (
                                <ShieldCheck className="h-3.5 w-3.5" aria-hidden />
                              )}
                              {claimed ? "Claim filed" : "File claim"}
                            </button>
                          </div>
                        </div>
                      </div>
                    </li>
                  );
                })}
              </ul>
            )}
          </div>
        </div>
      ) : null}
    </div>
  );
}

export default NotificationsPanel;
