from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
import os
import json
import logging
import uuid
from datetime import datetime, date, timedelta
from typing import Any
from contextlib import contextmanager
from dotenv import load_dotenv, find_dotenv
import psycopg
from psycopg.rows import dict_row
from groq import Groq

load_dotenv(find_dotenv())

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("predictive-analytics-agent")

app = FastAPI()

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

client = Groq(api_key=os.environ.get("GROQ_API_KEY"))

DATABASE_URL = os.environ.get("DATABASE_URL")
if not DATABASE_URL:
    raise RuntimeError("DATABASE_URL environment variable is not set")

MAX_AGENT_STEPS = 8


# ────────────────────────────────────────────────────────────────
#  DATABASE HELPERS
# ────────────────────────────────────────────────────────────────

@contextmanager
def get_conn():
    conn = psycopg.connect(DATABASE_URL, row_factory=dict_row)
    try:
        yield conn
        conn.commit()
    except Exception:
        conn.rollback()
        raise
    finally:
        conn.close()


def _serialize(obj: Any) -> Any:
    """Make DB result rows JSON-safe (datetime → ISO string, UUID → str)."""
    if isinstance(obj, datetime):
        return obj.isoformat()
    if isinstance(obj, date):
        return obj.isoformat()
    if isinstance(obj, uuid.UUID):
        return str(obj)
    raise TypeError(f"Object of type {type(obj)} is not JSON serializable")


def rows_to_json(rows: list) -> str:
    return json.dumps(rows, default=_serialize)


# ────────────────────────────────────────────────────────────────
#  TOOL DEFINITIONS  (Groq function-calling schemas)
# ────────────────────────────────────────────────────────────────

TOOL_DEFINITIONS = [
    {
        "type": "function",
        "function": {
            "name": "fetch_route_statistics",
            "description": (
                "Query the database for incident counts (lost reports + found items) "
                "grouped by route and station over a given time window. "
                "Returns ranked list of locations with incident totals."
            ),
            "parameters": {
                "type": "object",
                "properties": {
                    "days_back": {
                        "type": "integer",
                        "description": "How many days of history to analyse (e.g. 30, 90, 365)",
                    }
                },
                "required": ["days_back"],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "fetch_temporal_patterns",
            "description": (
                "Query the database for time-based loss patterns: "
                "breakdowns by day-of-week and by month. "
                "Reveals when incidents peak."
            ),
            "parameters": {
                "type": "object",
                "properties": {
                    "days_back": {
                        "type": "integer",
                        "description": "How many days of history to include",
                    }
                },
                "required": ["days_back"],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "fetch_category_hotspots",
            "description": (
                "Query the database for the item categories most commonly lost at each "
                "route or station. Helps identify whether a hotspot sees mostly electronics, "
                "bags, documents, etc."
            ),
            "parameters": {
                "type": "object",
                "properties": {
                    "days_back": {
                        "type": "integer",
                        "description": "How many days of history to include",
                    },
                    "top_n": {
                        "type": "integer",
                        "description": "Number of top locations to analyse (default 10)",
                    },
                },
                "required": ["days_back"],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "generate_hotspot_report",
            "description": (
                "Compile all gathered statistics into a final structured hotspot report. "
                "Assign a risk_score (0–10) and risk_level (low/medium/high/critical) to "
                "each location. Include trend direction and AI recommendations."
            ),
            "parameters": {
                "type": "object",
                "properties": {
                    "route_stats": {
                        "type": "string",
                        "description": "JSON string of route statistics from fetch_route_statistics",
                    },
                    "temporal_stats": {
                        "type": "string",
                        "description": "JSON string of temporal patterns from fetch_temporal_patterns",
                    },
                    "category_stats": {
                        "type": "string",
                        "description": "JSON string of category hotspots from fetch_category_hotspots",
                    },
                },
                "required": ["route_stats", "temporal_stats", "category_stats"],
            },
        },
    },
]


# ────────────────────────────────────────────────────────────────
#  TOOL IMPLEMENTATIONS  (execute real DB queries)
# ────────────────────────────────────────────────────────────────

def tool_fetch_route_statistics(days_back: int) -> dict:
    since = datetime.now() - timedelta(days=days_back)

    try:
        with get_conn() as conn:
            with conn.cursor() as cur:
                # Passenger lost reports grouped by route / station
                cur.execute(
                    """
                    SELECT
                        COALESCE(r.route_name, lr.route_or_station, 'Unknown') AS location,
                        r.id::text                                              AS route_id,
                        COUNT(lr.id)                                            AS incident_count,
                        COUNT(lr.id) FILTER (WHERE lr.status = 'open')         AS open_count,
                        COUNT(lr.id) FILTER (WHERE lr.status = 'matched')      AS matched_count,
                        MAX(lr.created_at)                                      AS last_incident
                    FROM lost_reports lr
                    LEFT JOIN routes r ON r.id = lr.route_id
                    WHERE lr.created_at >= %s
                    GROUP BY COALESCE(r.route_name, lr.route_or_station, 'Unknown'), r.id
                    ORDER BY incident_count DESC
                    LIMIT 20
                    """,
                    (since,),
                )
                rows = cur.fetchall()

                cur.execute(
                    "SELECT COUNT(*) AS n FROM lost_reports WHERE created_at >= %s",
                    (since,),
                )
                total = cur.fetchone()["n"]

    except Exception as e:
        logger.error(f"DB error in fetch_route_statistics: {e}")
        return {"error": str(e), "locations": [], "total_incidents": 0}

    locations = [
        {
            "location": row["location"],
            "route_id": row["route_id"],
            "incident_count": row["incident_count"],
            "open_count": row["open_count"],
            "matched_count": row["matched_count"],
            "last_incident": row["last_incident"],
        }
        for row in rows
    ]

    return {
        "period_days": days_back,
        "since": since.isoformat(),
        "total_incidents": total,
        "locations": locations,
    }


def tool_fetch_temporal_patterns(days_back: int) -> dict:
    since = datetime.now() - timedelta(days=days_back)

    try:
        with get_conn() as conn:
            with conn.cursor() as cur:
                # Day-of-week breakdown (0=Sunday in PostgreSQL DOW)
                cur.execute(
                    """
                    SELECT
                        TO_CHAR(created_at, 'Day') AS day_name,
                        EXTRACT(DOW FROM created_at)::int AS day_num,
                        COUNT(*) AS incident_count
                    FROM lost_reports
                    WHERE created_at >= %s
                    GROUP BY day_name, day_num
                    ORDER BY day_num
                    """,
                    (since,),
                )
                by_day = cur.fetchall()

                # Monthly breakdown
                cur.execute(
                    """
                    SELECT
                        TO_CHAR(created_at, 'YYYY-MM') AS month,
                        COUNT(*) AS incident_count
                    FROM lost_reports
                    WHERE created_at >= %s
                    GROUP BY month
                    ORDER BY month
                    """,
                    (since,),
                )
                by_month = cur.fetchall()

                # Hour-of-day breakdown (from date_lost field if populated, else created_at)
                cur.execute(
                    """
                    SELECT
                        EXTRACT(HOUR FROM created_at)::int AS hour_of_day,
                        COUNT(*) AS incident_count
                    FROM lost_reports
                    WHERE created_at >= %s
                    GROUP BY hour_of_day
                    ORDER BY hour_of_day
                    """,
                    (since,),
                )
                by_hour = cur.fetchall()

    except Exception as e:
        logger.error(f"DB error in fetch_temporal_patterns: {e}")
        return {"error": str(e), "by_day": [], "by_month": [], "by_hour": []}

    return {
        "period_days": days_back,
        "by_day_of_week": [dict(r) for r in by_day],
        "by_month": [dict(r) for r in by_month],
        "by_hour_of_day": [dict(r) for r in by_hour],
    }


def tool_fetch_category_hotspots(days_back: int, top_n: int = 10) -> dict:
    since = datetime.now() - timedelta(days=days_back)

    try:
        with get_conn() as conn:
            with conn.cursor() as cur:
                # Categories most lost per route/station
                cur.execute(
                    """
                    SELECT
                        COALESCE(r.route_name, lr.route_or_station, 'Unknown') AS location,
                        COALESCE(lr.category, 'Other')                          AS category,
                        COUNT(*)                                                 AS count
                    FROM lost_reports lr
                    LEFT JOIN routes r ON r.id = lr.route_id
                    WHERE lr.created_at >= %s
                    GROUP BY location, category
                    ORDER BY location, count DESC
                    LIMIT %s
                    """,
                    (since, top_n * 5),  # fetch more, group below
                )
                rows = cur.fetchall()

                # Overall category distribution
                cur.execute(
                    """
                    SELECT
                        COALESCE(category, 'Other') AS category,
                        COUNT(*) AS count
                    FROM lost_reports
                    WHERE created_at >= %s
                    GROUP BY category
                    ORDER BY count DESC
                    """,
                    (since,),
                )
                overall_categories = cur.fetchall()

    except Exception as e:
        logger.error(f"DB error in fetch_category_hotspots: {e}")
        return {"error": str(e), "by_location": [], "overall": []}

    # Group by location → top categories
    location_map: dict[str, list] = {}
    for row in rows:
        loc = row["location"]
        if loc not in location_map:
            location_map[loc] = []
        location_map[loc].append({"category": row["category"], "count": row["count"]})

    by_location = [
        {"location": loc, "top_categories": cats[:5]}
        for loc, cats in list(location_map.items())[:top_n]
    ]

    return {
        "period_days": days_back,
        "by_location": by_location,
        "overall_category_distribution": [dict(r) for r in overall_categories],
    }


def tool_generate_hotspot_report(
    route_stats: str, temporal_stats: str, category_stats: str
) -> dict:
    """Ask the LLM to interpret all statistics and produce the final structured report."""

    system_prompt = """You are a transit safety analytics AI for a lost & found system.
You have been given three data sources derived exclusively from passenger lost item reports:
1. route_stats: lost report counts per route/station (passenger submissions only)
2. temporal_stats: time-based patterns (day-of-week, month, hour)
3. category_stats: item categories most reported lost at each location

Your task: produce a structured JSON hotspot report.

Rules:
- risk_score: float 0.0–10.0 (higher = more risk)
- risk_level: "low" (<3), "medium" (3–5), "high" (6–8), "critical" (>8)
- trend: "increasing", "stable", or "decreasing" (infer from available data; default "stable" if unclear)
- Include up to 10 hotspots ranked by risk_score
- If total_incidents is 0, return an empty hotspots list with summary "No incidents recorded yet"

Return ONLY a JSON object in this exact format:
{
  "summary": "one-paragraph narrative for transit authorities",
  "total_incidents_analyzed": <integer>,
  "hotspots": [
    {
      "rank": 1,
      "location": "<route or station name>",
      "route_id": "<uuid or null>",
      "incident_count": <integer>,
      "lost_count": <integer>,
      "found_count": <integer>,
      "risk_score": <float>,
      "risk_level": "<low|medium|high|critical>",
      "trend": "<increasing|stable|decreasing>",
      "top_categories": ["<category>", ...],
      "recommendation": "<one actionable sentence>"
    }
  ],
  "temporal_insights": {
    "peak_day": "<day name or null>",
    "peak_hour_range": "<e.g. 07:00–09:00 or null>",
    "busiest_month": "<YYYY-MM or null>"
  },
  "recommendations": [
    "<overall recommendation 1>",
    "<overall recommendation 2>"
  ]
}"""

    messages = [
        {"role": "system", "content": system_prompt},
        {
            "role": "user",
            "content": (
                f"ROUTE STATISTICS:\n{route_stats}\n\n"
                f"TEMPORAL PATTERNS:\n{temporal_stats}\n\n"
                f"CATEGORY DISTRIBUTION:\n{category_stats}"
            ),
        },
    ]

    completion = client.chat.completions.create(
        model="llama-3.3-70b-versatile",
        messages=messages,
        temperature=0.2,
        max_tokens=2048,
    )
    reply = completion.choices[0].message.content.strip()
    return _parse_json(reply)


# ────────────────────────────────────────────────────────────────
#  AGENT — orchestrates the tool-use reasoning loop
# ────────────────────────────────────────────────────────────────

AGENT_SYSTEM_PROMPT = """You are the Predictive Analytics Agent for SmartFind, a transit lost & found system.

Your goal is to analyse historical lost-and-found data and generate a daily hotspot map for transit authorities.

You have four tools:
1. fetch_route_statistics(days_back)   — incident counts per route/station
2. fetch_temporal_patterns(days_back)  — when losses peak (day, month, hour)
3. fetch_category_hotspots(days_back)  — which item types are lost where
4. generate_hotspot_report(...)        — compile everything into the final report

STRATEGY:
- Step 1: Call fetch_route_statistics with days_back=90 for a broad view
- Step 2: Call fetch_temporal_patterns with days_back=90
- Step 3: Call fetch_category_hotspots with days_back=90 and top_n=10
- Step 4: Call generate_hotspot_report with the JSON results from all three steps

Always execute all four steps in order. Call one tool at a time."""


def run_analytics_agent() -> dict:
    """Run the predictive analytics agent loop and return the hotspot report."""

    messages = [
        {"role": "system", "content": AGENT_SYSTEM_PROMPT},
        {
            "role": "user",
            "content": (
                "Generate today's hotspot analysis report. "
                "Start with step 1: fetch route statistics."
            ),
        },
    ]

    # Collected raw stats — passed to generate_hotspot_report
    collected = {
        "route_stats": None,
        "temporal_stats": None,
        "category_stats": None,
    }
    final_report = None

    for step in range(MAX_AGENT_STEPS):
        logger.info(f"Analytics agent step {step + 1}/{MAX_AGENT_STEPS}")

        completion = client.chat.completions.create(
            model="llama-3.3-70b-versatile",
            messages=messages,
            tools=TOOL_DEFINITIONS,
            tool_choice="auto",
            temperature=0.1,
            max_tokens=1024,
        )

        response_message = completion.choices[0].message
        messages.append(response_message)

        if not response_message.tool_calls:
            logger.info("Agent finished — no more tool calls")
            break

        for tool_call in response_message.tool_calls:
            fn_name = tool_call.function.name
            fn_args = json.loads(tool_call.function.arguments)
            logger.info(f"Tool call: {fn_name}({fn_args})")

            if fn_name == "fetch_route_statistics":
                result = tool_fetch_route_statistics(fn_args.get("days_back", 90))
                tool_result_str = rows_to_json(result) if isinstance(result, list) else json.dumps(result, default=_serialize)
                collected["route_stats"] = tool_result_str

            elif fn_name == "fetch_temporal_patterns":
                result = tool_fetch_temporal_patterns(fn_args.get("days_back", 90))
                tool_result_str = json.dumps(result, default=_serialize)
                collected["temporal_stats"] = tool_result_str

            elif fn_name == "fetch_category_hotspots":
                result = tool_fetch_category_hotspots(
                    fn_args.get("days_back", 90),
                    fn_args.get("top_n", 10),
                )
                tool_result_str = json.dumps(result, default=_serialize)
                collected["category_stats"] = tool_result_str

            elif fn_name == "generate_hotspot_report":
                # Agent may pass data it collected via earlier tool results
                route_s = fn_args.get("route_stats") or collected["route_stats"] or "{}"
                temporal_s = fn_args.get("temporal_stats") or collected["temporal_stats"] or "{}"
                category_s = fn_args.get("category_stats") or collected["category_stats"] or "{}"
                result_dict = tool_generate_hotspot_report(route_s, temporal_s, category_s)
                final_report = result_dict
                tool_result_str = json.dumps(result_dict, default=_serialize)

            else:
                tool_result_str = f"Unknown tool: {fn_name}"

            messages.append({
                "role": "tool",
                "tool_call_id": tool_call.id,
                "content": tool_result_str,
            })

        if final_report is not None:
            logger.info("Analytics agent completed — hotspot report generated")
            break

    # Fallback: if agent never called generate_hotspot_report, run it directly
    if final_report is None:
        logger.warning("Agent did not call generate_hotspot_report — running fallback")
        route_s = collected["route_stats"] or json.dumps(tool_fetch_route_statistics(90), default=_serialize)
        temporal_s = collected["temporal_stats"] or json.dumps(tool_fetch_temporal_patterns(90), default=_serialize)
        category_s = collected["category_stats"] or json.dumps(tool_fetch_category_hotspots(90), default=_serialize)
        final_report = tool_generate_hotspot_report(route_s, temporal_s, category_s)

    return final_report


# ────────────────────────────────────────────────────────────────
#  PERSIST REPORT TO DATABASE
# ────────────────────────────────────────────────────────────────

def save_report(report: dict, report_date: date) -> str:
    """Upsert the hotspot report into the hotspot_reports table. Returns the row id."""
    record_id = str(uuid.uuid4())

    try:
        with get_conn() as conn:
            with conn.cursor() as cur:
                cur.execute(
                    """
                    INSERT INTO hotspot_reports
                        (id, report_date, generated_at, total_incidents,
                         hotspots, temporal_insights, category_distribution,
                         ai_summary, ai_recommendations)
                    VALUES (%s, %s, NOW(), %s, %s, %s, %s, %s, %s)
                    ON CONFLICT (report_date) DO UPDATE SET
                        generated_at         = NOW(),
                        total_incidents      = EXCLUDED.total_incidents,
                        hotspots             = EXCLUDED.hotspots,
                        temporal_insights    = EXCLUDED.temporal_insights,
                        category_distribution= EXCLUDED.category_distribution,
                        ai_summary           = EXCLUDED.ai_summary,
                        ai_recommendations   = EXCLUDED.ai_recommendations
                    RETURNING id
                    """,
                    (
                        record_id,
                        report_date,
                        report.get("total_incidents_analyzed", 0),
                        json.dumps(report.get("hotspots", [])),
                        json.dumps(report.get("temporal_insights", {})),
                        json.dumps({}),                              # reserved for future category breakdown
                        report.get("summary", ""),
                        json.dumps(report.get("recommendations", [])),
                    ),
                )
                row = cur.fetchone()
                return str(row["id"]) if row else record_id
    except Exception as e:
        logger.error(f"Failed to save hotspot report: {e}")
        raise


# ────────────────────────────────────────────────────────────────
#  HELPERS
# ────────────────────────────────────────────────────────────────

def _parse_json(text: str) -> dict:
    text = text.strip()
    if text.startswith("```"):
        lines = text.split("\n")
        text = "\n".join(lines[1:-1])
    return json.loads(text)


# ────────────────────────────────────────────────────────────────
#  PYDANTIC MODELS
# ────────────────────────────────────────────────────────────────

class RunAnalyticsRequest(BaseModel):
    report_date: str | None = None   # ISO date string, defaults to today


class HotspotEntry(BaseModel):
    rank: int
    location: str
    route_id: str | None
    incident_count: int
    lost_count: int
    found_count: int
    risk_score: float
    risk_level: str
    trend: str
    top_categories: list[str]
    recommendation: str


class TemporalInsights(BaseModel):
    peak_day: str | None
    peak_hour_range: str | None
    busiest_month: str | None


class AnalyticsReport(BaseModel):
    report_date: str
    generated_at: str
    summary: str
    total_incidents_analyzed: int
    hotspots: list[dict]
    temporal_insights: dict
    recommendations: list[str]


# ────────────────────────────────────────────────────────────────
#  API ENDPOINTS
# ────────────────────────────────────────────────────────────────

@app.get("/health")
def health():
    return "Predictive analytics agent is running!"


@app.post("/analytics/run")
def run_analytics(req: RunAnalyticsRequest):
    """
    Trigger the predictive analytics agent manually.

    This endpoint is also the target for a nightly cron job (e.g. 02:00 AM server time).
    The agent queries historical data, identifies hotspots, and persists the result.

    Returns the generated report immediately.
    """
    if not os.environ.get("GROQ_API_KEY"):
        raise HTTPException(status_code=500, detail="GROQ_API_KEY not configured")

    # Determine target date
    target_date: date
    if req.report_date:
        try:
            target_date = date.fromisoformat(req.report_date)
        except ValueError:
            raise HTTPException(status_code=400, detail="report_date must be ISO format: YYYY-MM-DD")
    else:
        target_date = date.today()

    logger.info(f"Running analytics agent for {target_date}")

    try:
        report = run_analytics_agent()
    except Exception as e:
        logger.error(f"Analytics agent failed: {e}")
        raise HTTPException(status_code=500, detail=f"Agent failed: {e}")

    # Persist to database (non-blocking on error — still return the report)
    try:
        report_id = save_report(report, target_date)
        logger.info(f"Hotspot report saved: id={report_id} date={target_date}")
    except Exception as e:
        logger.warning(f"Could not persist report: {e}")
        report_id = None

    return {
        "report_date": target_date.isoformat(),
        "generated_at": datetime.now().isoformat(),
        "report_id": report_id,
        **report,
    }


@app.get("/analytics/heatmap")
def get_heatmap(days: int = 7):
    """
    Return heatmap data for the latest stored hotspot report.
    Falls back to a live lightweight query if no report is stored yet,
    and returns an empty response if no data exists at all.

    Query param:
      days  — how many days of history to consider when falling back (default 7)
    """
    row = None
    try:
        with get_conn() as conn:
            with conn.cursor() as cur:
                cur.execute(
                    """
                    SELECT id::text, report_date, generated_at, total_incidents,
                           hotspots, temporal_insights, ai_summary, ai_recommendations
                    FROM hotspot_reports
                    ORDER BY report_date DESC
                    LIMIT 1
                    """,
                )
                row = cur.fetchone()
    except Exception as e:
        # Table may not exist yet (migration pending) — fall through to live query
        logger.warning(f"Could not query hotspot_reports (migration pending?): {e}")

    if row is not None:
        return {
            "source": "stored_report",
            "report_id": row["id"],
            "report_date": row["report_date"].isoformat() if row["report_date"] else None,
            "generated_at": row["generated_at"].isoformat() if row["generated_at"] else None,
            "total_incidents": row["total_incidents"],
            "hotspots": row["hotspots"],
            "temporal_insights": row["temporal_insights"],
            "summary": row["ai_summary"],
            "recommendations": row["ai_recommendations"],
        }

    # No stored report — try a lightweight live aggregation
    try:
        live = tool_fetch_route_statistics(days)
    except Exception as e:
        logger.warning(f"Live query also failed: {e}")
        return {
            "source": "unavailable",
            "message": "No incidents recorded yet. The heatmap will populate as data arrives.",
            "total_incidents": 0,
            "hotspots": [],
        }

    if live.get("total_incidents", 0) == 0:
        return {
            "source": "live",
            "message": "No incidents recorded yet. The heatmap will populate as data arrives.",
            "total_incidents": 0,
            "hotspots": [],
        }

    return {
        "source": "live",
        "period_days": days,
        "total_incidents": live["total_incidents"],
        "hotspots": live.get("locations", []),
    }


@app.get("/analytics/hotspots")
def get_hotspots(top: int = 10):
    """
    Return the top N high-risk routes/stations from the latest stored report.
    Falls back to a live aggregation if no report exists.
    """
    row = None
    try:
        with get_conn() as conn:
            with conn.cursor() as cur:
                cur.execute(
                    """
                    SELECT hotspots, report_date, ai_summary
                    FROM hotspot_reports
                    ORDER BY report_date DESC
                    LIMIT 1
                    """,
                )
                row = cur.fetchone()
    except Exception as e:
        # Table may not exist yet — fall through to live query
        logger.warning(f"Could not query hotspot_reports (migration pending?): {e}")

    if row is None:
        # No stored report — return live ranking
        try:
            live = tool_fetch_route_statistics(30)
        except Exception as e:
            raise HTTPException(status_code=500, detail=f"No stored report and live query failed: {e}")

        locations = live.get("locations", [])[:top]
        if not locations:
            return {
                "source": "live",
                "message": "No incident data available yet.",
                "hotspots": [],
            }
        return {
            "source": "live",
            "note": "Run POST /analytics/run to generate a full AI-powered report",
            "hotspots": locations,
        }

    hotspots = (row["hotspots"] or [])[:top]
    return {
        "source": "stored_report",
        "report_date": row["report_date"].isoformat() if row["report_date"] else None,
        "summary": row["ai_summary"],
        "hotspots": hotspots,
    }


@app.get("/analytics/history")
def get_history(limit: int = 30):
    """Return a list of past hotspot reports (metadata only, no full hotspot payload)."""
    rows = []
    try:
        with get_conn() as conn:
            with conn.cursor() as cur:
                cur.execute(
                    """
                    SELECT id::text, report_date, generated_at, total_incidents, ai_summary
                    FROM hotspot_reports
                    ORDER BY report_date DESC
                    LIMIT %s
                    """,
                    (limit,),
                )
                rows = cur.fetchall()
    except Exception as e:
        logger.warning(f"Could not query hotspot_reports history (migration pending?): {e}")

    return {
        "count": len(rows),
        "reports": [
            {
                "id": r["id"],
                "report_date": r["report_date"].isoformat() if r["report_date"] else None,
                "generated_at": r["generated_at"].isoformat() if r["generated_at"] else None,
                "total_incidents": r["total_incidents"],
                "summary": r["ai_summary"],
            }
            for r in rows
        ],
    }
