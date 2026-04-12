import { useEffect, useState } from "react";
import {
  AlertTriangle,
  BarChart3,
  Calendar,
  CircleDot,
  Clock,
  Flame,
  Lightbulb,
  Loader2,
  MapPin,
  Minus,
  RefreshCw,
  ShieldAlert,
  TrendingDown,
  TrendingUp,
} from "lucide-react";

const ANALYTICS_BASE_URL =
  import.meta.env.VITE_ANALYTICS_API_URL || "http://localhost:8092";

// ─── Risk thresholds (computed from relative incident density) ──
function computeRiskLevel(count, max) {
  if (max === 0) return "low";
  const ratio = count / max;
  if (ratio >= 0.75) return "critical";
  if (ratio >= 0.45) return "high";
  if (ratio >= 0.2) return "medium";
  return "low";
}

const RISK = {
  critical: {
    bar: "bg-red-500",
    badge: "bg-red-100 text-red-700 border-red-200",
    card: "border-red-100 bg-red-50/40",
    dot: "bg-red-500",
    score: "text-red-600",
    label: "Critical",
  },
  high: {
    bar: "bg-orange-500",
    badge: "bg-orange-100 text-orange-700 border-orange-200",
    card: "border-orange-100 bg-orange-50/40",
    dot: "bg-orange-500",
    score: "text-orange-600",
    label: "High",
  },
  medium: {
    bar: "bg-amber-400",
    badge: "bg-amber-100 text-amber-700 border-amber-200",
    card: "border-amber-100 bg-amber-50/30",
    dot: "bg-amber-400",
    score: "text-amber-600",
    label: "Medium",
  },
  low: {
    bar: "bg-green-500",
    badge: "bg-green-100 text-green-700 border-green-200",
    card: "border-green-100 bg-green-50/30",
    dot: "bg-green-500",
    score: "text-green-600",
    label: "Low",
  },
};

// ─── Helpers ────────────────────────────────────────────────────
function TrendIcon({ trend }) {
  if (trend === "increasing")
    return <TrendingUp className="h-3.5 w-3.5 text-red-500" />;
  if (trend === "decreasing")
    return <TrendingDown className="h-3.5 w-3.5 text-green-500" />;
  return <Minus className="h-3.5 w-3.5 text-muted-foreground/50" />;
}

function fmt(dateStr) {
  if (!dateStr) return "—";
  return new Date(dateStr).toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

// ─── Stat Card ──────────────────────────────────────────────────
function StatCard({ icon: Icon, label, value, accent, sub }) {
  return (
    <div className="rounded-2xl border border-border bg-card p-5 shadow-sm">
      <div className="flex items-start gap-3">
        <div
          className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-xl ${accent}`}
        >
          <Icon className="h-5 w-5" />
        </div>
        <div className="min-w-0 flex-1">
          <p className="text-2xl font-semibold tracking-tight">{value}</p>
          <p className="text-xs text-muted-foreground">{label}</p>
          {sub && (
            <p className="mt-0.5 truncate text-[11px] text-muted-foreground/60">
              {sub}
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

// ─── Hotspot Row ────────────────────────────────────────────────
function HotspotRow({ hotspot, maxCount, rank, isAI }) {
  const riskLevel = isAI
    ? hotspot.risk_level
    : computeRiskLevel(hotspot.incident_count ?? hotspot.total_incidents, maxCount);
  const risk = RISK[riskLevel] || RISK.low;
  const count = hotspot.incident_count ?? hotspot.total_incidents ?? 0;
  const pct = maxCount > 0 ? (count / maxCount) * 100 : 0;

  return (
    <div className={`rounded-2xl border p-4 transition hover:shadow-sm ${risk.card}`}>
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-center gap-3">
          <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-xl bg-foreground/8 text-xs font-semibold text-foreground/70">
            {rank}
          </span>
          <div className="min-w-0">
            <h3 className="truncate text-sm font-semibold">
              {hotspot.location}
            </h3>
            <p className="mt-0.5 text-[11px] text-muted-foreground">
              {count} lost report{count !== 1 ? "s" : ""}
              {hotspot.open_count != null && (
                <> · {hotspot.open_count} open</>
              )}
            </p>
          </div>
        </div>

        <div className="flex shrink-0 items-center gap-2">
          {isAI && <TrendIcon trend={hotspot.trend} />}
          <span
            className={`inline-flex items-center gap-1 rounded-full border px-2.5 py-0.5 text-[11px] font-semibold ${risk.badge}`}
          >
            <CircleDot className="h-2.5 w-2.5" />
            {risk.label}
          </span>
        </div>
      </div>

      {/* density bar */}
      <div className="mt-3 h-1.5 w-full overflow-hidden rounded-full bg-border/60">
        <div
          className={`h-full rounded-full ${risk.bar}`}
          style={{ width: `${pct}%` }}
        />
      </div>

      <div className="mt-3 flex flex-wrap items-center justify-between gap-2">
        {/* categories (AI reports only) */}
        {isAI && (hotspot.top_categories || []).length > 0 && (
          <div className="flex flex-wrap gap-1.5">
            {hotspot.top_categories.slice(0, 3).map((cat) => (
              <span
                key={cat}
                className="rounded-full bg-background/70 px-2 py-0.5 text-[10px] font-medium text-foreground/70 ring-1 ring-border"
              >
                {cat}
              </span>
            ))}
          </div>
        )}
        {/* risk score (AI) or relative density (live) */}
        <span className="ml-auto text-[11px] font-medium text-muted-foreground">
          {isAI ? (
            <>
              Score{" "}
              <span className={`font-semibold ${risk.score}`}>
                {hotspot.risk_score?.toFixed(1) ?? "—"}
              </span>
              /10
            </>
          ) : (
            <span className={`font-semibold ${risk.score}`}>
              {Math.round(pct)}% density
            </span>
          )}
        </span>
      </div>

      {isAI && hotspot.recommendation && (
        <p className="mt-2 text-[11px] leading-relaxed text-muted-foreground/80">
          <span className="font-medium text-foreground/60">Rec: </span>
          {hotspot.recommendation}
        </p>
      )}
    </div>
  );
}

// ─── Main Panel ─────────────────────────────────────────────────
export default function AnalyticsPanel() {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      setError(null);
      try {
        const res = await fetch(`${ANALYTICS_BASE_URL}/analytics/heatmap`);
        if (!res.ok) throw new Error(`Service returned ${res.status}`);
        const json = await res.json();
        if (!cancelled) setData(json);
      } catch (err) {
        if (!cancelled) setError(err.message);
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    load();
    return () => { cancelled = true; };
  }, []);

  // Normalise: stored AI reports use data.hotspots; live queries use data.hotspots or data.locations
  const isAI = data?.source === "stored_report";
  const hotspots = data?.hotspots ?? data?.locations ?? [];
  const totalIncidents =
    data?.total_incidents_analyzed ?? data?.total_incidents ?? 0;
  const maxCount = hotspots.reduce(
    (m, h) => Math.max(m, h.incident_count ?? h.total_incidents ?? 0),
    0,
  );
  const criticalCount = hotspots.filter((h) => {
    const lvl = isAI
      ? h.risk_level
      : computeRiskLevel(h.incident_count ?? h.total_incidents, maxCount);
    return lvl === "critical";
  }).length;
  const highRiskCount = hotspots.filter((h) => {
    const lvl = isAI
      ? h.risk_level
      : computeRiskLevel(h.incident_count ?? h.total_incidents, maxCount);
    return lvl === "critical" || lvl === "high";
  }).length;

  const isEmpty =
    !loading && !error && (hotspots.length === 0 || totalIncidents === 0);

  // ── Loading ────────────────────────────────────────────────────
  if (loading) {
    return (
      <div className="flex items-center justify-center py-24 text-muted-foreground">
        <Loader2 className="mr-2 h-5 w-5 animate-spin" />
        <span className="text-sm">Loading hotspot data…</span>
      </div>
    );
  }

  // ── Error ──────────────────────────────────────────────────────
  if (error) {
    return (
      <div className="rounded-2xl border border-red-200 bg-red-50 p-6 text-sm text-red-700">
        <p className="font-semibold">Could not reach the analytics service</p>
        <p className="mt-1 text-red-600/80">{error}</p>
      </div>
    );
  }

  // ── Empty ──────────────────────────────────────────────────────
  if (isEmpty) {
    return (
      <div className="rounded-2xl border border-dashed border-border bg-card p-12 text-center shadow-sm">
        <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-muted">
          <BarChart3 className="h-7 w-7 text-muted-foreground/40" />
        </div>
        <h3 className="mt-4 text-sm font-semibold">No hotspot data yet</h3>
        <p className="mx-auto mt-1.5 max-w-xs text-xs leading-relaxed text-muted-foreground">
          {data?.message ||
            "Hotspot data will appear here automatically as passengers submit lost item reports."}
        </p>
      </div>
    );
  }

  // ── Data ───────────────────────────────────────────────────────
  return (
    <div className="space-y-6">
      {/* page label */}
      <div>
        <p className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
          Predictive Analytics
        </p>
        <h2 className="mt-1 text-xl font-semibold tracking-tight">
          Hotspot Map
        </h2>
        <p className="mt-0.5 text-sm text-muted-foreground">
          High-risk routes and stations based on passenger lost item reports.
        </p>
      </div>

      {/* stats */}
      <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
        <StatCard
          icon={BarChart3}
          label="Total Reports"
          value={totalIncidents}
          accent="bg-foreground/10 text-foreground"
        />
        <StatCard
          icon={ShieldAlert}
          label="High-Risk Zones"
          value={highRiskCount}
          accent="bg-orange-100 text-orange-700"
        />
        <StatCard
          icon={Flame}
          label="Critical Hotspots"
          value={criticalCount}
          accent="bg-red-100 text-red-700"
        />
        <StatCard
          icon={Clock}
          label="Last Updated"
          value={data?.generated_at ? fmt(data.generated_at) : "Live"}
          accent="bg-muted text-muted-foreground"
          sub={data?.report_date ? `Report: ${data.report_date}` : undefined}
        />
      </div>

      {/* main grid */}
      <div className="grid gap-6 lg:grid-cols-5">
        {/* hotspot list — 3 cols */}
        <div className="space-y-4 lg:col-span-3">
          {/* legend */}
          <div className="flex flex-wrap gap-2">
            {["critical", "high", "medium", "low"].map((lvl) => {
              const r = RISK[lvl];
              return (
                <span
                  key={lvl}
                  className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-[11px] font-medium ${r.badge}`}
                >
                  <span className={`h-1.5 w-1.5 rounded-full ${r.dot}`} />
                  {r.label}
                </span>
              );
            })}
            <span className="ml-auto text-[11px] text-muted-foreground">
              {hotspots.length} location{hotspots.length !== 1 ? "s" : ""}
            </span>
          </div>

          <div className="space-y-3">
            {hotspots.map((h, i) => (
              <HotspotRow
                key={(h.location || h.route_id || "") + i}
                hotspot={h}
                maxCount={maxCount}
                rank={h.rank ?? i + 1}
                isAI={isAI}
              />
            ))}
          </div>
        </div>

        {/* insights panel — 2 cols */}
        <div className="space-y-4 lg:col-span-2">
          {/* AI summary */}
          {data?.summary && (
            <div className="rounded-2xl border border-border bg-card p-5 shadow-sm">
              <p className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
                AI Summary
              </p>
              <p className="mt-3 text-sm leading-relaxed text-foreground/80">
                {data.summary}
              </p>
            </div>
          )}

          {/* Temporal insights */}
          {data?.temporal_insights && (
            <div className="rounded-2xl border border-border bg-card p-5 shadow-sm">
              <div className="flex items-center gap-2">
                <Calendar className="h-4 w-4 text-muted-foreground" />
                <p className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
                  Temporal Patterns
                </p>
              </div>
              <div className="mt-4 space-y-2">
                {[
                  { label: "Peak day", value: data.temporal_insights.peak_day, icon: Calendar },
                  { label: "Peak hours", value: data.temporal_insights.peak_hour_range, icon: Clock },
                  { label: "Busiest month", value: data.temporal_insights.busiest_month, icon: BarChart3 },
                ].map(({ label, value, icon: Icon }) => (
                  <div
                    key={label}
                    className="flex items-center justify-between rounded-xl bg-muted/50 px-3.5 py-2.5"
                  >
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <Icon className="h-3.5 w-3.5" />
                      {label}
                    </div>
                    <span className="text-xs font-semibold">{value || "—"}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Recommendations */}
          {(data?.recommendations?.length > 0) && (
            <div className="rounded-2xl border border-border bg-card p-5 shadow-sm">
              <div className="flex items-center gap-2">
                <Lightbulb className="h-4 w-4 text-amber-500" />
                <p className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
                  Recommendations
                </p>
              </div>
              <ul className="mt-4 space-y-2.5">
                {data.recommendations.map((rec, i) => (
                  <li key={i} className="flex items-start gap-2.5 text-xs leading-relaxed">
                    <span className="mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded-full bg-amber-100 text-[9px] font-bold text-amber-700">
                      {i + 1}
                    </span>
                    <span className="text-muted-foreground">{rec}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {/* source badge */}
          <div className="flex items-center justify-between rounded-xl border border-border/60 bg-muted/30 px-3.5 py-2.5">
            <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
              <MapPin className="h-3 w-3" />
              Based on
            </div>
            <span className="rounded-full bg-background px-2 py-0.5 text-[10px] font-medium ring-1 ring-border">
              Passenger lost reports
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}
