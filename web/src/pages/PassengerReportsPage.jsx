import { useEffect, useState } from "react";
import { passengerListLostReports } from "../api/gateway";

export default function PassengerReportsPage() {
  const [status, setStatus] = useState("open");
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
          <select
            value={status}
            onChange={(e) => setStatus(e.target.value)}
            className="h-9 rounded-xl border border-border bg-background px-3 text-sm"
          >
            <option value="open">Open</option>
            <option value="matched">Matched</option>
            <option value="closed">Closed</option>
            <option value="">All</option>
          </select>
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
          <div className="space-y-3">
            {reports.map((r) => (
              <div
                key={r.id}
                className="rounded-2xl border border-border/80 bg-card p-4 shadow-sm"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-semibold">
                      {r.item_name || "Unnamed item"}
                    </p>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {r.route_or_station || "—"} •{" "}
                      {r.date_lost ? new Date(r.date_lost).toLocaleString() : "—"}
                    </p>
                    <p className="mt-1 text-xs text-muted-foreground">
                      Report ID: <span className="font-mono">{r.id}</span>
                    </p>
                  </div>
                  <span className="inline-flex shrink-0 items-center rounded-full border border-border px-2 py-0.5 text-xs font-medium">
                    {r.status || "open"}
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

