import { useEffect, useState } from "react";
import { passengerListLostReports } from "../api/gateway";

function formatDate(value) {
  if (!value) return "Not provided";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Not provided";
  return date.toLocaleString();
}

function statusBadgeClass(status) {
  const normalized = String(status || "open").toLowerCase();
  if (normalized === "matched") {
    return "border-emerald-200/80 bg-emerald-50/80 text-emerald-700";
  }
  if (normalized === "closed") {
    return "border-slate-200/80 bg-slate-100/80 text-slate-700";
  }
  return "border-amber-200/80 bg-amber-50/80 text-amber-700";
}

export default function PassengerReportsPage() {
  const [status, setStatus] = useState("");
  const [loading, setLoading] = useState(false);
  const [reports, setReports] = useState([]);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      setError("");
      try {
        const res = await passengerListLostReports({ status });
        if (!cancelled) setReports(res?.reports || []);
      } catch (e) {
        if (!cancelled) setError(e?.message || "Failed to load lost reports.");
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
          <h1 className="text-base font-semibold">Lost reports</h1>
          <div className="relative">
            <label htmlFor="report-status" className="sr-only">
              Filter reports by status
            </label>
            <select
              id="report-status"
              value={status}
              onChange={(e) => setStatus(e.target.value)}
              className="h-9 appearance-none rounded-xl border border-border/80 bg-background pl-3 pr-9 text-sm text-foreground shadow-sm outline-none transition focus:border-border focus:ring-2 focus:ring-ring/30"
            >
              <option value="">All statuses</option>
              <option value="open">Open</option>
              <option value="matched">Matched</option>
              <option value="closed">Closed</option>
            </select>
            <span
              aria-hidden
              className="pointer-events-none absolute inset-y-0 right-3 flex items-center text-muted-foreground"
            >
              ▾
            </span>
          </div>
        </div>
        <p className="mt-1 text-sm text-muted-foreground">
          View your filed lost item reports.
        </p>
      </header>

      <div className="flex-1 overflow-y-auto py-4">
        {loading ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : error ? (
          <p className="text-sm text-red-600">{error}</p>
        ) : reports.length === 0 ? (
          <p className="text-sm text-muted-foreground">No reports found.</p>
        ) : (
          <div className="grid gap-3 sm:grid-cols-2">
            {reports.map((r) => (
              <div
                key={r.id}
                className="rounded-2xl border border-border/70 bg-card/95 p-4 transition hover:bg-card"
              >
                <div className="flex items-start justify-between gap-2">
                  <p className="truncate pr-2 text-sm font-semibold sm:text-base">
                    {r.item_name || "Unnamed item"}
                  </p>
                  <span
                    className={`inline-flex shrink-0 items-center rounded-full border px-2 py-0.5 text-xs font-semibold capitalize ${statusBadgeClass(r.status)}`}
                  >
                    {r.status || "open"}
                  </span>
                </div>

                <dl className="mt-3 space-y-1.5 text-xs sm:text-sm">
                  <div className="flex items-start justify-between gap-3">
                    <dt className="text-muted-foreground">Route/Station</dt>
                    <dd className="text-right text-foreground">
                      {r.route_or_station || "Not provided"}
                    </dd>
                  </div>
                  <div className="flex items-start justify-between gap-3">
                    <dt className="text-muted-foreground">Date Lost</dt>
                    <dd className="text-right text-foreground">
                      {formatDate(r.date_lost)}
                    </dd>
                  </div>
                </dl>

                {r.item_description ? (
                  <div className="mt-3 rounded-lg border border-border/60 bg-background/80 px-3 py-2">
                    <p className="text-[11px] uppercase tracking-wide text-muted-foreground">
                      Description
                    </p>
                    <p className="mt-1 line-clamp-3 text-xs text-foreground sm:text-sm">
                      {r.item_description}
                    </p>
                  </div>
                ) : null}

                {(r.color || r.brand || r.model) && (
                  <div className="mt-3 flex flex-wrap gap-1.5">
                    {r.color ? (
                      <span className="rounded-full border border-border/70 bg-muted/30 px-2 py-0.5 text-[11px] text-muted-foreground">
                        {r.color}
                      </span>
                    ) : null}
                    {r.brand ? (
                      <span className="rounded-full border border-border/70 bg-muted/30 px-2 py-0.5 text-[11px] text-muted-foreground">
                        {r.brand}
                      </span>
                    ) : null}
                    {r.model ? (
                      <span className="rounded-full border border-border/70 bg-muted/30 px-2 py-0.5 text-[11px] text-muted-foreground">
                        {r.model}
                      </span>
                    ) : null}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
