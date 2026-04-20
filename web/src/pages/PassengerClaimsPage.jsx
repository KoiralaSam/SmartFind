import { useEffect, useState } from "react";
import { passengerListClaims } from "../api/gateway";

export default function PassengerClaimsPage() {
  const [status, setStatus] = useState("");
  const [loading, setLoading] = useState(false);
  const [claims, setClaims] = useState([]);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      setError("");
      try {
        const res = await passengerListClaims({ status, limit: 100, offset: 0 });
        if (!cancelled) setClaims(res?.claims || []);
      } catch (e) {
        if (!cancelled) setError(e?.message || "Failed to load claims.");
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    load();
    return () => {
      cancelled = true;
    };
  }, [status]);

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
            {claims.map((c) => (
              <div
                key={c.id}
                className="rounded-2xl border border-border/80 bg-card p-4 shadow-sm"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-semibold">
                      Claim ID: <span className="font-mono">{c.id}</span>
                    </p>
                    <p className="mt-1 text-xs text-muted-foreground">
                      Found item: <span className="font-mono">{c.item_id}</span>
                    </p>
                    {c.lost_report_id ? (
                      <p className="mt-1 text-xs text-muted-foreground">
                        Lost report:{" "}
                        <span className="font-mono">{c.lost_report_id}</span>
                      </p>
                    ) : null}
                    {c.message ? (
                      <p className="mt-2 text-xs text-muted-foreground/90">
                        {c.message}
                      </p>
                    ) : null}
                  </div>
                  <span className="inline-flex shrink-0 items-center rounded-full border border-border px-2 py-0.5 text-xs font-medium">
                    {c.status || "pending"}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

