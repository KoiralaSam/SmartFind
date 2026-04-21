import { useEffect, useState } from "react";
import { MapContainer, TileLayer, CircleMarker, useMap } from "react-leaflet";
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
  X,
} from "lucide-react";

const ANALYTICS_BASE_URL = import.meta.env.VITE_ANALYTICS_API_URL || "";

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
function extractPrimaryLocation(name) {
  const parts = name.split(/\s*(?:->|→|–|—|\bto\b)\s*/i);
  return (parts[0] || name).trim();
}

async function geocode(name) {
  const query = extractPrimaryLocation(name);
  try {
    const res = await fetch(
      `https://nominatim.openstreetmap.org/search?q=${encodeURIComponent(query + ", USA")}&format=json&limit=1&countrycodes=us`,
      { headers: { "User-Agent": "SmartFind-LostFound/1.0" } }
    );
    const data = await res.json();
    if (data?.length > 0)
      return [parseFloat(data[0].lat), parseFloat(data[0].lon)];
  } catch {
    // Best-effort geocoding; fallback handled by returning null.
  }
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

// ─── SVG Line Chart ─────────────────────────────────────────────
function DayLineChart({ byDay, reports }) {
  const W = 560;
  const H = 210;
  const PAD = { top: 20, right: 20, bottom: 36, left: 50 };
  const chartW = W - PAD.left - PAD.right;
  const chartH = H - PAD.top - PAD.bottom;

  if (!byDay || byDay.length === 0) {
    return (
      <div className="flex h-[210px] items-center justify-center text-xs text-muted-foreground">
        No passenger data yet
      </div>
    );
  }

  const Y_MIN = 0;
  const Y_MAX = 23;
  const n = byDay.length; // always 7
  const xStep = chartW / (n - 1);
  const yScale = (h) => PAD.top + chartH - ((h - Y_MIN) / (Y_MAX - Y_MIN)) * chartH;
  const xAt = (i) => PAD.left + i * xStep;

  // Y ticks
  const yTicks = [0, 6, 12, 18, 23];

  // Line uses avg_hour; days with no data → treat as null, draw gap-free by
  // using 0 so the line stays continuous (user request).
  const linePts = byDay.map((d, i) => ({
    x: xAt(i),
    y: yScale(d.avg_hour != null ? d.avg_hour : 0),
    hasData: d.avg_hour != null,
  }));
  const polyline = linePts.map((p) => `${p.x},${p.y}`).join(" ");

  // Individual report scatter dots
  const scatterDots = (reports || []).map((r) => ({
    x: xAt(r.day_num),
    y: yScale(r.hour),
  }));

  return (
    <svg viewBox={`0 0 ${W} ${H}`} width="100%" style={{ overflow: "visible" }}>
      {/* grid lines */}
      {yTicks.map((t) => (
        <line
          key={t}
          x1={PAD.left}
          x2={PAD.left + chartW}
          y1={yScale(t)}
          y2={yScale(t)}
          stroke="#e5e7eb"
          strokeWidth="1"
        />
      ))}

      {/* y-axis labels */}
      {yTicks.map((t) => (
        <text key={t} x={PAD.left - 6} y={yScale(t) + 4} fontSize="9" fill="#9ca3af" textAnchor="end">
          {String(t).padStart(2, "0")}:00
        </text>
      ))}

      {/* continuous average line — spans all 7 days */}
      <polyline
        points={polyline}
        fill="none"
        stroke="#3b82f6"
        strokeWidth="2.5"
        strokeLinejoin="round"
        strokeLinecap="round"
        opacity="0.35"
      />

      {/* individual report dots (exact time each passenger submitted) */}
      {scatterDots.map((p, i) => (
        <circle
          key={i}
          cx={p.x}
          cy={p.y}
          r="5"
          fill="#3b82f6"
          stroke="#fff"
          strokeWidth="2"
        />
      ))}

      {/* x-axis day labels */}
      {byDay.map((d, i) => (
        <text
          key={d.day}
          x={xAt(i)}
          y={H - 4}
          fontSize="10"
          fill="#6b7280"
          textAnchor="middle"
        >
          {d.day}
        </text>
      ))}
    </svg>
  );
}

// ─── Helpers ────────────────────────────────────────────────────
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
function StatCard({ icon, label, value, accent }) {
  const IconComponent = icon;
  return (
    <div className="rounded-2xl border border-border bg-card p-3 shadow-sm">
      <div className="flex items-start gap-2.5">
        <div
          className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-xl ${accent}`}
        >
          {IconComponent ? <IconComponent className="h-4 w-4" /> : null}
        </div>
        <div className="min-w-0 flex-1">
          <p className="text-xl font-semibold tracking-tight">{value}</p>
          <p className="text-xs text-muted-foreground">{label}</p>
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
  const [mapped, setMapped] = useState([]);
  const [geocoding, setGeocoding] = useState(false);
  const [selectedLocation, setSelectedLocation] = useState(null);
  const [temporalByDay, setTemporalByDay] = useState(null);
  const [temporalReports, setTemporalReports] = useState(null);

  // Fetch heatmap — re-runs every 60s
  useEffect(() => {
    let cancelled = false;
    async function load() {
      setError(null);
      try {
        const res = await fetch(`${ANALYTICS_BASE_URL}/analytics/heatmap`);
        if (!res.ok) throw new Error(`Service returned ${res.status}`);
        const json = await res.json();
        if (!cancelled) {
          setData(json);
          setLoading(false);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err.message);
          setLoading(false);
        }
      }
    }
    load();
    const interval = setInterval(load, 60000);
    return () => { cancelled = true; clearInterval(interval); };
  }, []);

  // Fetch temporal data — re-runs every 60s to pick up new lost reports
  useEffect(() => {
    let cancelled = false;
    async function loadTemporal() {
      try {
        const res = await fetch(`${ANALYTICS_BASE_URL}/analytics/temporal`);
        if (!res.ok) return;
        const json = await res.json();
        if (!cancelled) {
          setTemporalByDay(json.by_day_of_week ?? null);
          setTemporalReports(json.reports ?? null);
        }
      } catch {
        // Non-blocking: keep previous temporal data if refresh fails.
      }
    }
    loadTemporal();
    const interval = setInterval(loadTemporal, 60000);
    return () => { cancelled = true; clearInterval(interval); };
  }, []);

  // Geocode hotspots when data arrives
  useEffect(() => {
    if (!data) return;
    const hotspots = data.hotspots ?? data.locations ?? [];
    if (!hotspots.length) { setMapped([]); return; }

    let cancelled = false;
    async function geocodeAll() {
      setGeocoding(true);
      const results = [];
      for (const h of hotspots) {
        if (cancelled) break;
        const coords = await geocode(h.location);
        results.push({ hotspot: h, coords });
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

  const rawHotspots = data?.hotspots ?? data?.locations ?? [];
  const totalIncidents =
    data?.total_incidents_analyzed ?? data?.total_incidents ?? 0;
  const maxCount = rawHotspots.reduce(
    (m, h) => Math.max(m, h.incident_count ?? h.total_incidents ?? 0),
    0
  );
  const hotspots = rawHotspots;
  const overallRecs = data?.recommendations ?? [];
  const criticalCount = hotspots.filter(
    (h) => computeRiskLevel(h.incident_count ?? h.total_incidents ?? 0, maxCount) === "critical"
  ).length;
  const highRiskCount = hotspots.filter((h) => {
    const lvl = computeRiskLevel(h.incident_count ?? h.total_incidents ?? 0, maxCount);
    return lvl === "critical" || lvl === "high";
  }).length;

  const placedCoords = mapped.filter((m) => m.coords).map((m) => m.coords);
  const isEmpty = !loading && !error && (hotspots.length === 0 || totalIncidents === 0);

  // Filtered recommendations based on selected dot
  const withRec = hotspots.filter((h) => h.recommendation);
  const filteredRecs = selectedLocation
    ? withRec.filter((h) => h.location === selectedLocation)
    : withRec;

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
            "Hotspot data will appear here automatically as data arrives."}
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
        <h2 className="mt-1 text-xl font-semibold tracking-tight">Hotspot Map</h2>
        <p className="mt-0.5 text-sm text-muted-foreground">
          High-risk routes and stations based on found item reports.
        </p>
      </div>

      {/* stats */}
      <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
        <StatCard icon={BarChart3} label="Total Reports" value={totalIncidents} accent="bg-foreground/10 text-foreground" />
        <StatCard icon={ShieldAlert} label="High-Risk Zones" value={highRiskCount} accent="bg-orange-100 text-orange-700" />
        <StatCard icon={Flame} label="Critical Hotspots" value={criticalCount} accent="bg-red-100 text-red-700" />
        <StatCard icon={Clock} label="Last Updated" value={data?.generated_at ? fmt(data.generated_at) : "Live"} accent="bg-muted text-muted-foreground" />
      </div>

      {/* ── MAP ── */}
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
          <span className="ml-auto text-[11px] text-muted-foreground">
            {selectedLocation
              ? `Showing: ${selectedLocation} — click map to clear`
              : "Click a dot to filter recommendations"}
          </span>
          {geocoding && (
            <span className="flex items-center gap-1 text-[11px] text-muted-foreground">
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
          // Click on map background clears selection
          eventHandlers={{ click: () => setSelectedLocation(null) }}
        >
          <TileLayer
            attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
            url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
          />

          {placedCoords.length > 0 && <FitBounds coords={placedCoords} />}

          {mapped.map(({ hotspot: h, coords }, i) => {
            if (!coords) return null;
            const count = h.incident_count ?? h.total_incidents ?? 0;
            const lvl = computeRiskLevel(count, maxCount);
            const risk = RISK[lvl] || RISK.low;
            const isSelected = selectedLocation === h.location;

            return (
              <CircleMarker
                key={(h.location || h.route_id || "") + i}
                center={coords}
                radius={risk.radius}
                pathOptions={{
                  color: isSelected ? "#1d4ed8" : risk.color,
                  fillColor: risk.fill,
                  fillOpacity: isSelected ? 1 : 0.75,
                  weight: isSelected ? 4 : 2,
                }}
                eventHandlers={{
                  click: (e) => {
                    e.originalEvent.stopPropagation();
                    setSelectedLocation(
                      isSelected ? null : h.location
                    );
                  },
                }}
              />
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

      {/* ── Staff Recommendations ── */}
      {(filteredRecs.length > 0 || (!selectedLocation && overallRecs.length > 0)) && (
        <div className="rounded-2xl border border-amber-200 bg-amber-50/40 p-5 shadow-sm">
          <div className="flex items-center gap-2 mb-4">
            <Lightbulb className="h-4 w-4 text-amber-500" />
            <p className="text-xs font-semibold uppercase tracking-[0.18em] text-amber-700">
              {selectedLocation ? `Recommendations — ${selectedLocation}` : "Staff Recommendations"}
            </p>
            {selectedLocation && (
              <button
                onClick={() => setSelectedLocation(null)}
                className="ml-auto flex items-center gap-1 rounded-full bg-amber-100 px-2 py-0.5 text-[11px] text-amber-700 hover:bg-amber-200"
              >
                <X className="h-3 w-3" /> Show all
              </button>
            )}
          </div>

          {/* per-route rows */}
          {filteredRecs.length > 0 && (
            <div className="space-y-2 mb-4">
              {filteredRecs.map((h, i) => {
                const count = h.incident_count ?? h.total_incidents ?? 0;
                const lvl = computeRiskLevel(count, maxCount);
                const risk = RISK[lvl] || RISK.low;
                return (
                  <div
                    key={h.location + i}
                    className="flex items-start gap-3 rounded-xl bg-white/70 px-4 py-3 text-xs shadow-sm"
                  >
                    <span
                      className="mt-0.5 flex h-2.5 w-2.5 shrink-0 rounded-full"
                      style={{ background: risk.fill }}
                    />
                    <div className="min-w-0">
                      <span className="font-semibold text-foreground/80">{h.location}</span>
                      <span className="mx-1.5 text-muted-foreground/40">·</span>
                      <span className="text-amber-900/70">{h.recommendation}</span>
                    </div>
                  </div>
                );
              })}
            </div>
          )}

          {/* overall recs — only when no location selected */}
          {!selectedLocation && overallRecs.length > 0 && (
            <>
              <p className="mb-2 text-[11px] font-medium uppercase tracking-widest text-amber-600/70">
                Overall Actions
              </p>
              <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
                {overallRecs.map((rec, i) => (
                  <div
                    key={i}
                    className="flex items-start gap-2.5 rounded-xl bg-white/60 px-4 py-3 text-xs shadow-sm"
                  >
                    <span className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-amber-400 text-[9px] font-bold text-white">
                      {i + 1}
                    </span>
                    <span className="text-amber-900/80">{rec}</span>
                  </div>
                ))}
              </div>
            </>
          )}
        </div>
      )}

      {/* ── Temporal Line Chart ── */}
      <div className="rounded-2xl border border-border bg-card p-5 shadow-sm">
        <div className="flex items-center gap-2 mb-4">
          <Calendar className="h-4 w-4 text-muted-foreground" />
          <p className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
            Avg Time of Day Lost — by Day of Week
          </p>
          <span className="ml-auto text-[11px] text-muted-foreground">Passenger data · Y = hour (00–23)</span>
        </div>
        <DayLineChart byDay={temporalByDay} reports={temporalReports} />
      </div>
    </div>
  );
}
