import { useEffect, useState } from "react";
import { useLocation } from "react-router-dom";
import { passengerListClaims } from "../api/gateway";

const POLL_INTERVAL_MS = 30000;

function formatDateLabel(v) {
  if (v == null || v === "") return "";
  try {
    const d = new Date(v);
    if (Number.isNaN(d.getTime())) return "";
    return d.toLocaleDateString(undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  } catch {
    return "";
  }
}

export default function PassengerClaimsPage() {
  const location = useLocation();
  const [status, setStatus] = useState("");
  const [loading, setLoading] = useState(false);
  const [claims, setClaims] = useState([]);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    let timer = null;

    function mergeClaims(nextClaims) {
      setClaims((prev) => {
        const prevById = new Map(prev.map((c) => [c.id, c]));
        return (Array.isArray(nextClaims) ? nextClaims : []).map((c) => {
          const prevClaim = prevById.get(c.id);
          if (c?.found_item) return c;
          if (!prevClaim?.found_item) return c;
          const next = { ...c, found_item: { ...prevClaim.found_item } };
          const claimStatus = String(c?.status || "").toLowerCase();
          if (claimStatus === "approved") {
            next.found_item.status = "claimed";
          }
          return next;
        });
      });
    }

    async function load(showSpinner) {
      if (showSpinner) setLoading(true);
      setError("");
      try {
        const res = await passengerListClaims({ status, limit: 100, offset: 0 });
        if (!cancelled) mergeClaims(res?.claims || []);
      } catch (e) {
        if (!cancelled) setError(e?.message || "Failed to load claims.");
      } finally {
        if (!cancelled && showSpinner) setLoading(false);
      }
    }

    load(true);
    timer = setInterval(() => {
      load(false);
    }, POLL_INTERVAL_MS);

    return () => {
      cancelled = true;
      if (timer) clearInterval(timer);
    };
  }, [status, location.pathname, location.key]);

  return (
    <div className="mx-auto flex h-full max-w-4xl flex-col overflow-hidden px-3 py-4 sm:px-4">
      <header className="shrink-0 border-b border-border/80 pb-3">
        <div className="flex items-center justify-between gap-3">
          <h1 className="text-base font-semibold">Claims</h1>
          <select
            value={status}
            onChange={(e) => setStatus(e.target.value)}
            className="h-9 rounded-xl border border-border bg-background px-3 text-sm"
          >
            <option value="">All</option>
            <option value="pending">Pending</option>
            <option value="approved">Approved</option>
            <option value="rejected">Rejected</option>
            <option value="cancelled">Cancelled</option>
          </select>
        </div>
        <p className="mt-1 text-sm text-muted-foreground">
          View the claims you’ve filed on found items.
        </p>
      </header>

      <div className="flex-1 overflow-y-auto py-4">
        {loading ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : error ? (
          <p className="text-sm text-red-600">{error}</p>
        ) : claims.length === 0 ? (
          <p className="text-sm text-muted-foreground">No claims found.</p>
        ) : (
          <div className="space-y-3">
            {claims.map((c) => {
              if (!c) return null;
              const fi = c.found_item;
              if (!fi) {
                return (
                  <div
                    key={c.id}
                    className="rounded-2xl border border-border/80 bg-card p-4 shadow-sm"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <p className="truncate text-sm font-semibold">
                          Claim filed
                        </p>
                        <p className="mt-1 text-xs text-muted-foreground">
                          Found item:{" "}
                          <span className="font-mono">{c.item_id}</span>
                        </p>
                        {c.message ? (
                          <p className="mt-2 text-xs text-muted-foreground/90">
                            {c.message}
                          </p>
                        ) : null}
                      </div>
                      <span className="inline-flex shrink-0 items-center rounded-full border border-border px-2 py-0.5 text-xs font-medium capitalize">
                        {c.status || "pending"}
                      </span>
                    </div>
                  </div>
                );
              }
              const primary =
                fi.primary_image_url ||
                (Array.isArray(fi.image_urls) ? fi.image_urls[0] : null);
              const thumbs =
                Array.isArray(fi.image_urls) && fi.image_urls.length > 1
                  ? fi.image_urls.slice(1, 5)
                  : [];
              const title =
                (fi.item_name && String(fi.item_name).trim()) || "Found item";
              const dateFoundStr = formatDateLabel(fi.date_found);
              const metaLine = [
                fi.category,
                fi.color && fi.color !== "unknown" ? fi.color : null,
                fi.brand && fi.brand !== "unknown" ? fi.brand : null,
                fi.location_found,
              ]
                .filter(Boolean)
                .join(" · ");
              const displayStatus =
                String(fi.status || "").toLowerCase() === "claimed"
                  ? "claimed"
                  : c.status || "pending";
              return (
                <div
                  key={c.id}
                  className="rounded-2xl border border-border/80 bg-card p-5 shadow-sm"
                >
                  <div className="flex items-start gap-4">
                    {primary ? (
                      <div className="shrink-0">
                        <img
                          src={primary}
                          alt={title}
                          className="h-16 w-16 rounded-xl border border-border object-cover"
                        />
                        {thumbs.length > 0 ? (
                          <div className="mt-2 flex flex-wrap items-center gap-1">
                            {thumbs.map((src) => (
                              <img
                                key={src}
                                src={src}
                                alt=""
                                className="h-6 w-6 rounded-md border border-border object-cover"
                                loading="lazy"
                              />
                            ))}
                          </div>
                        ) : null}
                      </div>
                    ) : null}
                    <div className="flex min-w-0 flex-1 items-start justify-between gap-3">
                      <div className="min-w-0 space-y-1">
                        <h3 className="truncate text-sm font-semibold">
                          {title}
                        </h3>
                        {metaLine ? (
                          <p className="text-xs text-muted-foreground">
                            {metaLine}
                          </p>
                        ) : null}
                        {fi.item_description ? (
                          <p className="line-clamp-2 text-xs leading-relaxed text-muted-foreground">
                            {fi.item_description}
                          </p>
                        ) : null}
                        <p className="text-[11px] text-muted-foreground/70">
                          {dateFoundStr
                            ? `Found ${dateFoundStr}`
                            : "Date found unavailable"}
                          {fi.route_or_station
                            ? ` — ${fi.route_or_station}`
                            : ""}
                        </p>
                        {c.message ? (
                          <p className="mt-1 text-xs text-muted-foreground/90">
                            Your note: {c.message}
                          </p>
                        ) : null}
                      </div>
                      <span className="inline-flex shrink-0 items-center rounded-full border border-border px-2 py-0.5 text-xs font-medium capitalize">
                        {displayStatus}
                      </span>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

