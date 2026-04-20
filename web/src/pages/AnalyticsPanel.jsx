import { useEffect, useState } from "react";
import { MapContainer, TileLayer, CircleMarker, Tooltip, useMap } from "react-leaflet";
import "leaflet/dist/leaflet.css";
import {
  BarChart3,
  Calendar,
  Clock,
  Flame,
  Lightbulb,
  Loader2,
  MapPin,
  ShieldAlert,
} from "lucide-react";

const ANALYTICS_BASE_URL =
  import.meta.env.VITE_ANALYTICS_API_URL || "http://localhost:8092";

// ─── Risk config ────────────────────────────────────────────────
const RISK = {
  critical: {
    color: "#ef4444",
    fill: "#ef4444",
    radius: 20,
    label: "Critical",
    badge: "bg-red-100 text-red-700 border-red-200",
    dot: "bg-red-500",
  },
  high: {
    color: "#f97316",
    fill: "#f97316",
    radius: 15,
    label: "High",
    badge: "bg-orange-100 text-orange-700 border-orange-200",
    dot: "bg-orange-500",
  },
  medium: {
    color: "#f59e0b",
    fill: "#f59e0b",
    radius: 11,
    label: "Medium",
    badge: "bg-amber-100 text-amber-700 border-amber-200",
    dot: "bg-amber-400",
  },
  low: {
    color: "#22c55e",
    fill: "#22c55e",
    radius: 7,
    label: "Low",
    badge: "bg-green-100 text-green-700 border-green-200",
    dot: "bg-green-500",
  },
};

function computeRiskLevel(count, max) {
  if (max === 0) return "low";
  const r = count / max;
  if (r >= 0.75) return "critical";
  if (r >= 0.45) return "high";
  if (r >= 0.2) return "medium";
  return "low";
}

// ─── Geocode via Nominatim ───────────────────────────────────────
async function geocode(name) {
  try {
    const res = await fetch(
      `https://nominatim.openstreetmap.org/search?q=${encodeURIComponent(name + ", USA")}&format=json&limit=1&countrycodes=us`,
      { headers: { "User-Agent": "SmartFind-LostFound/1.0" } }
    );
    const data = await res.json();
    if (data?.length > 0)
      return [parseFloat(data[0].lat), parseFloat(data[0].lon)];
  } catch {}
  return null;
}

// ─── Fit map to markers ─────────────────────────────────────────
function FitBounds({ coords }) {
  const map = useMap();
  useEffect(() => {
    if (!coords.length) return;
    if (coords.length === 1) {
      map.setView(coords[0], 11);
    } else {
      map.fitBounds(coords, { padding: [50, 50] });
    }
  }, [map, coords]);
  return null;
}

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

// ─── Main Panel ─────────────────────────────────────────────────
export default function AnalyticsPanel() {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  // [{hotspot, coords: [lat, lng] | null}]
  const [mapped, setMapped] = useState([]);
  const [geocoding, setGeocoding] = useState(false);

  // Fetch heatmap data
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

  // Geocode hotspots when data arrives
  useEffect(() => {
    if (!data) return;
    const isAI = data.source === "stored_report";
    const hotspots = data.hotspots ?? data.locations ?? [];
    if (!hotspots.length) { setMapped([]); return; }

    let cancelled = false;
    async function geocodeAll() {
      setGeocoding(true);
      const results = [];
      for (const h of hotspots) {
        if (cancelled) break;
        const coords = await geocode(h.location);
        results.push({ hotspot: h, coords, isAI });
        // Rate-limit Nominatim: 1 req/sec
        await new Promise((r) => setTimeout(r, 1100));
      }
      if (!cancelled) {
        setMapped(results);
        setGeocoding(false);
      }
    }
    geocodeAll();
    return () => { cancelled = true; };
  }, [data]);

  const isAI = data?.source === "stored_report";
  const hotspots = data?.hotspots ?? data?.locations ?? [];
  const totalIncidents =
    data?.total_incidents_analyzed ?? data?.total_incidents ?? 0;
  const maxCount = hotspots.reduce(
    (m, h) => Math.max(m, h.incident_count ?? h.total_incidents ?? 0),
    0
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

  const placedCoords = mapped.filter((m) => m.coords).map((m) => m.coords);

  const isEmpty =
    !loading && !error && (hotspots.length === 0 || totalIncidents === 0);

  // ── Loading ─────────────────────────────────────────────────
  if (loading) {
    return (
      <div className="flex items-center justify-center py-24 text-muted-foreground">
        <Loader2 className="mr-2 h-5 w-5 animate-spin" />
        <span className="text-sm">Loading hotspot data…</span>
      </div>
    );
  }

  // ── Error ───────────────────────────────────────────────────
  if (error) {
    return (
      <div className="rounded-2xl border border-red-200 bg-red-50 p-6 text-sm text-red-700">
        <p className="font-semibold">Could not reach the analytics service</p>
        <p className="mt-1 text-red-600/80">{error}</p>
      </div>
    );
  }

  // ── Empty ───────────────────────────────────────────────────
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

  // ── Data ────────────────────────────────────────────────────
  return (
    <div className="space-y-6">
      {/* header */}
      <div>
        <p className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
          Predictive Analytics
        </p>
        <h2 className="mt-1 text-xl font-semibold tracking-tight">
          Hotspot Map
        </h2>
        <p className="mt-0.5 text-sm text-muted-foreground">
          High-risk routes and stations based on lost &amp; found item reports.
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

      {/* ── MAP — full width centered ── */}
      <div className="overflow-hidden rounded-2xl border border-border shadow-sm" style={{ height: 500 }}>
        {/* legend bar */}
        <div className="flex items-center gap-4 border-b border-border bg-card px-4 py-2.5">
          {["critical", "high", "medium", "low"].map((lvl) => {
            const r = RISK[lvl];
            return (
              <span
                key={lvl}
                className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-[11px] font-medium ${r.badge}`}
              >
                <span className={`h-2 w-2 rounded-full ${r.dot}`} />
                {r.label}
              </span>
            );
          })}
          {geocoding && (
            <span className="ml-auto flex items-center gap-1 text-[11px] text-muted-foreground">
              <Loader2 className="h-3 w-3 animate-spin" />
              Placing pins…
            </span>
          )}
        </div>

        <MapContainer
          center={[39.5, -98.35]}
          zoom={4}
          minZoom={3}
          maxBounds={[[15, -170], [75, -50]]}
          maxBoundsViscosity={1.0}
          style={{ height: "calc(100% - 41px)", width: "100%" }}
          scrollWheelZoom
        >
          <TileLayer
            attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
            url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
          />

          {placedCoords.length > 0 && <FitBounds coords={placedCoords} />}

          {mapped.map(({ hotspot: h, coords, isAI: ai }, i) => {
            if (!coords) return null;
            const count = h.incident_count ?? h.total_incidents ?? 0;
            const lvl = ai ? h.risk_level : computeRiskLevel(count, maxCount);
            const risk = RISK[lvl] || RISK.low;

            return (
              <CircleMarker
                key={(h.location || h.route_id || "") + i}
                center={coords}
                radius={risk.radius}
                pathOptions={{
                  color: risk.color,
                  fillColor: risk.fill,
                  fillOpacity: 0.75,
                  weight: 2,
                }}
              >
                <Tooltip direction="top" offset={[0, -risk.radius]} opacity={1}>
                  <div className="min-w-[160px] space-y-1 text-xs">
                    <p className="font-semibold">{h.location}</p>
                    <p>{count} lost report{count !== 1 ? "s" : ""}</p>
                    {h.open_count != null && (
                      <p className="text-muted-foreground">{h.open_count} open</p>
                    )}
                    <span
                      style={{
                        display: "inline-block",
                        background: risk.fill,
                        color: "#fff",
                        borderRadius: 999,
                        padding: "1px 8px",
                        fontSize: 10,
                        fontWeight: 600,
                      }}
                    >
                      {risk.label}
                    </span>
                    {ai && h.recommendation && (
                      <p className="mt-1 italic text-gray-600">{h.recommendation}</p>
                    )}
                  </div>
                </Tooltip>
              </CircleMarker>
            );
          })}
        </MapContainer>
      </div>

      {/* unmapped notice */}
      {mapped.some((m) => !m.coords) && (
        <p className="text-[11px] text-muted-foreground">
          <MapPin className="mr-1 inline h-3 w-3" />
          {mapped.filter((m) => !m.coords).length} location
          {mapped.filter((m) => !m.coords).length !== 1 ? "s" : ""} could not be placed on the map.
        </p>
      )}

      {/* ── Insights row — 3 cols ── */}
      <div className="grid gap-4 md:grid-cols-3">
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

        {/* Staff recommendations */}
        {data?.recommendations?.length > 0 && (
          <div className="rounded-2xl border border-amber-200 bg-amber-50/40 p-5 shadow-sm">
            <div className="flex items-center gap-2">
              <Lightbulb className="h-4 w-4 text-amber-500" />
              <p className="text-xs font-medium uppercase tracking-[0.18em] text-amber-700">
                Staff Action Plan
              </p>
            </div>
            <ul className="mt-4 space-y-3">
              {data.recommendations.map((rec, i) => (
                <li key={i} className="flex items-start gap-2.5 text-xs leading-relaxed">
                  <span className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-amber-400 text-[9px] font-bold text-white">
                    {i + 1}
                  </span>
                  <span className="text-amber-900/80">{rec}</span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </div>
  );
}
