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
from google.oauth2 import service_account
from google.cloud import bigquery
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

groq_client = Groq(api_key=os.environ.get("GROQ_API_KEY"))

DATABASE_URL = os.environ.get("DATABASE_URL")
if not DATABASE_URL:
    raise RuntimeError("DATABASE_URL environment variable is not set")

MAX_AGENT_STEPS = 8

# BigQuery project / dataset resolved from service account JSON
_sa_info: dict = {}
_raw_sa_json = os.environ.get("ANALYTICS_GOOGLE_SERVICE_ACCOUNT_JSON")
if _raw_sa_json:
    _sa_info = json.loads(_raw_sa_json)

BQ_PROJECT = os.environ.get("BIGQUERY_PROJECT") or _sa_info.get("project_id", "")
BQ_DATASET = os.environ.get("BIGQUERY_DATASET", "smartfind")


# BigQuery client

_bq_client: bigquery.Client | None = None


def get_bq_client() -> bigquery.Client:
    global _bq_client
    if _bq_client is None:
        if not _sa_info:
            raise RuntimeError("ANALYTICS_GOOGLE_SERVICE_ACCOUNT_JSON environment variable is not set")
        credentials = service_account.Credentials.from_service_account_info(
            _sa_info,
            scopes=["https://www.googleapis.com/auth/bigquery"],
        )
        _bq_client = bigquery.Client(project=BQ_PROJECT, credentials=credentials)
    return _bq_client


def _table(name: str) -> str:
    return f"`{BQ_PROJECT}.{BQ_DATASET}.{name}`"


# PostgreSQL helpers

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
    """Make result rows JSON-safe (datetime → ISO string, UUID → str)."""
    if isinstance(obj, datetime):
        return obj.isoformat()
    if isinstance(obj, date):
        return obj.isoformat()
    if isinstance(obj, uuid.UUID):
        return str(obj)
    raise TypeError(f"Object of type {type(obj)} is not JSON serializable")


def _parse_json(text: str) -> dict:
    text = text.strip()
    if text.startswith("```"):
        lines = text.split("\n")
        text = "\n".join(lines[1:-1])
    return json.loads(text)


# PostgreSQL → BigQuery sync

_LOST_REPORTS_SCHEMA = [
    bigquery.SchemaField("id", "STRING", mode="REQUIRED"),
    bigquery.SchemaField("route_id", "STRING", mode="NULLABLE"),
    bigquery.SchemaField("route_or_station", "STRING", mode="NULLABLE"),
    bigquery.SchemaField("category", "STRING", mode="NULLABLE"),
    bigquery.SchemaField("status", "STRING", mode="REQUIRED"),
    bigquery.SchemaField("created_at", "TIMESTAMP", mode="REQUIRED"),
]

_ROUTES_SCHEMA = [
    bigquery.SchemaField("id", "STRING", mode="REQUIRED"),
    bigquery.SchemaField("route_name", "STRING", mode="REQUIRED"),
    bigquery.SchemaField("created_at", "TIMESTAMP", mode="REQUIRED"),
]


def sync_to_bigquery() -> None:
    """
    Sync lost_reports and routes from PostgreSQL to BigQuery (full replace).
    Analytics is based on passenger lost reports only — not staff found items.
    """
    bq = get_bq_client()

    with get_conn() as conn:
        with conn.cursor() as cur:
            cur.execute(
                """
                SELECT id::text, route_id::text, route_or_station,
                       category, status::text, created_at
                FROM lost_reports
                """
            )
            lr_rows = [
                {
                    "id": r["id"],
                    "route_id": r["route_id"],
                    "route_or_station": r["route_or_station"],
                    "category": r["category"],
                    "status": r["status"],
                    "created_at": r["created_at"].isoformat() if r["created_at"] else None,
                }
                for r in cur.fetchall()
            ]

            cur.execute("SELECT id::text, route_name, created_at FROM routes")
            routes_rows = [
                {
                    "id": r["id"],
                    "route_name": r["route_name"],
                    "created_at": r["created_at"].isoformat() if r["created_at"] else None,
                }
                for r in cur.fetchall()
            ]

    lr_job = bq.load_table_from_json(
        lr_rows,
        f"{BQ_PROJECT}.{BQ_DATASET}.lost_reports",
        job_config=bigquery.LoadJobConfig(
            schema=_LOST_REPORTS_SCHEMA,
            write_disposition=bigquery.WriteDisposition.WRITE_TRUNCATE,
        ),
    )
    lr_job.result()

    routes_job = bq.load_table_from_json(
        routes_rows,
        f"{BQ_PROJECT}.{BQ_DATASET}.routes",
        job_config=bigquery.LoadJobConfig(
            schema=_ROUTES_SCHEMA,
            write_disposition=bigquery.WriteDisposition.WRITE_TRUNCATE,
        ),
    )
    routes_job.result()

    logger.info(
        f"Synced to BigQuery: {len(lr_rows)} lost_reports, {len(routes_rows)} routes"
    )


# BigQuery data fetches

def fetch_route_statistics(days_back: int) -> dict:
    since = datetime.now() - timedelta(days=days_back)
    bq = get_bq_client()

    job_config = bigquery.QueryJobConfig(
        query_parameters=[
            bigquery.ScalarQueryParameter("since", "TIMESTAMP", since),
        ]
    )

    query = f"""
        SELECT
            COALESCE(r.route_name, lr.route_or_station, 'Unknown') AS location,
            lr.route_id                                             AS route_id,
            COUNT(lr.id)                                            AS incident_count,
            COUNTIF(lr.status = 'open')                            AS open_count,
            COUNTIF(lr.status = 'matched')                         AS matched_count,
            MAX(lr.created_at)                                      AS last_incident
        FROM {_table('lost_reports')} lr
        LEFT JOIN {_table('routes')} r ON r.id = lr.route_id
        WHERE lr.created_at >= @since
        GROUP BY location, route_id
        ORDER BY incident_count DESC
        LIMIT 20
    """

    total_query = f"""
        SELECT COUNT(*) AS n
        FROM {_table('lost_reports')}
        WHERE created_at >= @since
    """

    rows = list(bq.query(query, job_config=job_config).result())
    total_rows = list(bq.query(total_query, job_config=job_config).result())
    total = total_rows[0]["n"] if total_rows else 0

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


def fetch_temporal_patterns(days_back: int) -> dict:
    since = datetime.now() - timedelta(days=days_back)
    bq = get_bq_client()

    job_config = bigquery.QueryJobConfig(
        query_parameters=[
            bigquery.ScalarQueryParameter("since", "TIMESTAMP", since),
        ]
    )

    # DAYOFWEEK: 1=Sunday … 7=Saturday in BigQuery
    dow_query = f"""
        SELECT
            FORMAT_TIMESTAMP('%A', created_at) AS day_name,
            EXTRACT(DAYOFWEEK FROM created_at)  AS day_num,
            COUNT(*)                            AS incident_count
        FROM {_table('lost_reports')}
        WHERE created_at >= @since
        GROUP BY day_name, day_num
        ORDER BY day_num
    """

    month_query = f"""
        SELECT
            FORMAT_TIMESTAMP('%Y-%m', created_at) AS month,
            COUNT(*)                              AS incident_count
        FROM {_table('lost_reports')}
        WHERE created_at >= @since
        GROUP BY month
        ORDER BY month
    """

    hour_query = f"""
        SELECT
            EXTRACT(HOUR FROM created_at) AS hour_of_day,
            COUNT(*)                      AS incident_count
        FROM {_table('lost_reports')}
        WHERE created_at >= @since
        GROUP BY hour_of_day
        ORDER BY hour_of_day
    """

    by_day = [dict(r) for r in bq.query(dow_query, job_config=job_config).result()]
    by_month = [dict(r) for r in bq.query(month_query, job_config=job_config).result()]
    by_hour = [dict(r) for r in bq.query(hour_query, job_config=job_config).result()]

    return {
        "period_days": days_back,
        "by_day_of_week": by_day,
        "by_month": by_month,
        "by_hour_of_day": by_hour,
    }


def fetch_category_hotspots(days_back: int, top_n: int = 10) -> dict:
    since = datetime.now() - timedelta(days=days_back)
    bq = get_bq_client()

    job_config = bigquery.QueryJobConfig(
        query_parameters=[
            bigquery.ScalarQueryParameter("since", "TIMESTAMP", since),
            bigquery.ScalarQueryParameter("limit_n", "INT64", top_n * 5),
        ]
    )

    location_query = f"""
        SELECT
            COALESCE(r.route_name, lr.route_or_station, 'Unknown') AS location,
            COALESCE(lr.category, 'Other')                          AS category,
            COUNT(*)                                                AS count
        FROM {_table('lost_reports')} lr
        LEFT JOIN {_table('routes')} r ON r.id = lr.route_id
        WHERE lr.created_at >= @since
        GROUP BY location, category
        ORDER BY location, count DESC
        LIMIT @limit_n
    """

    overall_query = f"""
        SELECT
            COALESCE(category, 'Other') AS category,
            COUNT(*)                    AS count
        FROM {_table('lost_reports')}
        WHERE created_at >= @since
        GROUP BY category
        ORDER BY count DESC
    """

    rows = list(bq.query(location_query, job_config=job_config).result())
    overall_categories = [dict(r) for r in bq.query(overall_query, job_config=job_config).result()]

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
        "overall_category_distribution": overall_categories,
    }


# Report generator (Groq LLM interprets BigQuery results)

def generate_hotspot_report(
    route_stats: dict,
    temporal_stats: dict,
    category_stats: dict,
) -> dict:
    """Send BigQuery results to Groq and get a structured hotspot report back."""

    system_prompt = """You are a transit safety analytics AI for a lost & found system.
You have been given three data sources queried from BigQuery:
1. route_stats: lost report counts per route/station
2. temporal_stats: time-based patterns (day-of-week, month, hour)
3. category_stats: item categories most reported lost at each location

Your task: produce a structured JSON hotspot report WITH actionable staff recommendations.

Rules:
- risk_score: float 0.0–10.0 (higher = more risk)
- risk_level: "low" (<3), "medium" (3–5), "high" (6–8), "critical" (>8)
- trend: "increasing", "stable", or "decreasing" (infer from data; default "stable" if unclear)
- Include up to 10 hotspots ranked by risk_score
- If total_incidents is 0, return an empty hotspots list with summary "No incidents recorded yet"
- For each hotspot recommendation: give specific, actionable advice for transit staff
  (e.g. deploy additional staff during peak hours, install lost item collection boxes,
  add signage, increase platform surveillance, run passenger awareness campaigns)
- For overall recommendations: include staffing strategies, patrol schedules, and prevention measures

Return ONLY valid JSON in this exact format:
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
      "recommendation": "<specific actionable staff instruction>"
    }
  ],
  "temporal_insights": {
    "peak_day": "<day name or null>",
    "peak_hour_range": "<e.g. 07:00–09:00 or null>",
    "busiest_month": "<YYYY-MM or null>"
  },
  "recommendations": [
    "<staff deployment or operational recommendation 1>",
    "<staff deployment or operational recommendation 2>",
    "<prevention or passenger awareness recommendation 3>"
  ]
}"""

    completion = groq_client.chat.completions.create(
        model="llama-3.3-70b-versatile",
        messages=[
            {"role": "system", "content": system_prompt},
            {
                "role": "user",
                "content": (
                    f"ROUTE STATISTICS:\n{json.dumps(route_stats, default=_serialize)}\n\n"
                    f"TEMPORAL PATTERNS:\n{json.dumps(temporal_stats, default=_serialize)}\n\n"
                    f"CATEGORY DISTRIBUTION:\n{json.dumps(category_stats, default=_serialize)}"
                ),
            },
        ],
        temperature=0.2,
        max_tokens=2048,
    )
    return _parse_json(completion.choices[0].message.content.strip())


# Analytics runner

def run_analytics() -> dict:
    """Sync PostgreSQL → BigQuery, then use Groq to generate the hotspot report."""
    logger.info("Syncing PostgreSQL data to BigQuery")
    sync_to_bigquery()

    logger.info("Fetching route statistics from BigQuery")
    route_stats = fetch_route_statistics(90)

    logger.info("Fetching temporal patterns from BigQuery")
    temporal_stats = fetch_temporal_patterns(90)

    logger.info("Fetching category hotspots from BigQuery")
    category_stats = fetch_category_hotspots(90, top_n=10)

    logger.info("Generating hotspot report with Groq")
    return generate_hotspot_report(route_stats, temporal_stats, category_stats)


# Persist report to PostgreSQL

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
                        json.dumps({}),  # reserved for future category breakdown
                        report.get("summary", ""),
                        json.dumps(report.get("recommendations", [])),
                    ),
                )
                row = cur.fetchone()
                return str(row["id"]) if row else record_id
    except Exception as e:
        logger.error(f"Failed to save hotspot report: {e}")
        raise


# Pydantic models

class RunAnalyticsRequest(BaseModel):
    report_date: str | None = None  # ISO date string, defaults to today


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


# API endpoints

@app.get("/health")
def health():
    return "Predictive analytics agent is running!"


@app.post("/analytics/run")
def run_analytics_endpoint(req: RunAnalyticsRequest):
    """
    Trigger the analytics agent manually.

    Syncs PostgreSQL data to BigQuery, queries BigQuery for historical patterns,
    uses Groq to generate the hotspot report, and persists the result to PostgreSQL.
    """
    if not os.environ.get("GROQ_API_KEY"):
        raise HTTPException(status_code=500, detail="GROQ_API_KEY not configured")
    if not _sa_info:
        raise HTTPException(status_code=500, detail="ANALYTICS_GOOGLE_SERVICE_ACCOUNT_JSON not configured")

    target_date: date
    if req.report_date:
        try:
            target_date = date.fromisoformat(req.report_date)
        except ValueError:
            raise HTTPException(status_code=400, detail="report_date must be ISO format: YYYY-MM-DD")
    else:
        target_date = date.today()

    logger.info(f"Running analytics for {target_date}")

    try:
        report = run_analytics()
    except Exception as e:
        logger.error(f"Analytics failed: {e}")
        raise HTTPException(status_code=500, detail=f"Analytics failed: {e}")

    # Persist to PostgreSQL (non-blocking on error — still return the report)
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
def get_heatmap(days: int = 90):
    """
    Return heatmap data from the latest stored hotspot report.
    If new found_items were added after the last report was generated,
    the cache is invalidated and a fresh report is generated.
    Falls back to a live PostgreSQL query if no report is stored yet.
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

                # Invalidate cache if new found_items arrived after last report
                if row is not None:
                    cur.execute("SELECT MAX(created_at) AS latest FROM found_items")
                    latest_row = cur.fetchone()
                    latest_item = latest_row["latest"] if latest_row else None
                    if latest_item and row["generated_at"] and latest_item > row["generated_at"]:
                        logger.info(
                            f"New found_items since last report ({latest_item} > {row['generated_at']})"
                            " — invalidating cache"
                        )
                        row = None
    except Exception as e:
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

    # No stored report — query PostgreSQL then run Groq to generate recommendations
    if not os.environ.get("GROQ_API_KEY"):
        return {
            "source": "unavailable",
            "message": "GROQ_API_KEY not configured.",
            "total_incidents": 0,
            "hotspots": [],
        }

    try:
        live_stats = _fetch_route_stats_postgres(days)
    except Exception as e:
        logger.warning(f"Live PostgreSQL query failed: {e}")
        return {
            "source": "unavailable",
            "message": "No incidents recorded yet. The heatmap will populate as data arrives.",
            "total_incidents": 0,
            "hotspots": [],
        }

    if live_stats.get("total_incidents", 0) == 0:
        return {
            "source": "live",
            "message": "No incidents recorded yet. The heatmap will populate as data arrives.",
            "total_incidents": 0,
            "hotspots": [],
        }

    # Generate Groq report from live PostgreSQL data and persist it
    try:
        logger.info("No stored report found — generating live Groq report from PostgreSQL")
        report = run_analytics_postgres()
        try:
            save_report(report, date.today())
            logger.info("Live Groq report saved to hotspot_reports")
        except Exception as save_err:
            logger.warning(f"Could not persist live report: {save_err}")
        return {
            "source": "stored_report",
            "report_date": date.today().isoformat(),
            "generated_at": datetime.now().isoformat(),
            "total_incidents": report.get("total_incidents_analyzed", live_stats["total_incidents"]),
            "hotspots": report.get("hotspots", []),
            "temporal_insights": report.get("temporal_insights", {}),
            "summary": report.get("summary", ""),
            "recommendations": report.get("recommendations", []),
        }
    except Exception as e:
        logger.warning(f"Groq report generation failed: {e}")
        return {
            "source": "live",
            "period_days": days,
            "total_incidents": live_stats["total_incidents"],
            "hotspots": live_stats.get("locations", []),
        }


def _fetch_route_stats_postgres(days_back: int) -> dict:
    """Fetch found_items counts per route/station from PostgreSQL."""
    with get_conn() as conn:
        with conn.cursor() as cur:
            cur.execute(
                """
                SELECT
                    COALESCE(r.route_name, fi.route_or_station, fi.location_found, 'Unknown') AS location,
                    fi.route_id::text                                                           AS route_id,
                    COUNT(fi.id)                                                                AS incident_count,
                    COUNT(fi.id) FILTER (WHERE fi.status = 'unclaimed')                        AS open_count,
                    COUNT(fi.id) FILTER (WHERE fi.status = 'claimed')                          AS matched_count,
                    MAX(fi.created_at)                                                          AS last_incident
                FROM found_items fi
                LEFT JOIN routes r ON r.id = fi.route_id
                GROUP BY location, fi.route_id
                ORDER BY incident_count DESC
                LIMIT 20
                """
            )
            rows = cur.fetchall()
            cur.execute("SELECT COUNT(*) AS n FROM found_items")
            total_row = cur.fetchone()
            total = total_row["n"] if total_row else 0

    return {
        "period_days": days_back,
        "total_incidents": total,
        "locations": [
            {
                "location": r["location"],
                "route_id": r["route_id"],
                "incident_count": r["incident_count"],
                "open_count": r["open_count"],
                "matched_count": r["matched_count"],
                "last_incident": r["last_incident"],
            }
            for r in rows
        ],
    }


def _fetch_temporal_patterns_postgres(days_back: int) -> dict:
    """Fetch time-based patterns from found_items in PostgreSQL."""
    since = datetime.now() - timedelta(days=days_back)
    with get_conn() as conn:
        with conn.cursor() as cur:
            cur.execute(
                """
                SELECT TO_CHAR(created_at, 'Day') AS day_name,
                       EXTRACT(DOW FROM created_at)::int AS day_num,
                       COUNT(*) AS incident_count
                FROM found_items WHERE created_at >= %s
                GROUP BY day_name, day_num ORDER BY day_num
                """,
                (since,),
            )
            by_day = [dict(r) for r in cur.fetchall()]

            cur.execute(
                """
                SELECT TO_CHAR(created_at, 'YYYY-MM') AS month,
                       COUNT(*) AS incident_count
                FROM found_items WHERE created_at >= %s
                GROUP BY month ORDER BY month
                """,
                (since,),
            )
            by_month = [dict(r) for r in cur.fetchall()]

            cur.execute(
                """
                SELECT EXTRACT(HOUR FROM created_at)::int AS hour_of_day,
                       COUNT(*) AS incident_count
                FROM found_items WHERE created_at >= %s
                GROUP BY hour_of_day ORDER BY hour_of_day
                """,
                (since,),
            )
            by_hour = [dict(r) for r in cur.fetchall()]

    return {
        "period_days": days_back,
        "by_day_of_week": by_day,
        "by_month": by_month,
        "by_hour_of_day": by_hour,
    }


def _fetch_category_hotspots_postgres(days_back: int, top_n: int = 10) -> dict:
    """Fetch category distribution per location from found_items in PostgreSQL."""
    since = datetime.now() - timedelta(days=days_back)
    with get_conn() as conn:
        with conn.cursor() as cur:
            cur.execute(
                """
                SELECT COALESCE(r.route_name, fi.route_or_station, fi.location_found, 'Unknown') AS location,
                       COALESCE(fi.category, 'Other') AS category,
                       COUNT(*) AS count
                FROM found_items fi
                LEFT JOIN routes r ON r.id = fi.route_id
                WHERE fi.created_at >= %s
                GROUP BY location, category
                ORDER BY location, count DESC
                LIMIT %s
                """,
                (since, top_n * 5),
            )
            rows = cur.fetchall()

            cur.execute(
                """
                SELECT COALESCE(category, 'Other') AS category, COUNT(*) AS count
                FROM found_items WHERE created_at >= %s
                GROUP BY category ORDER BY count DESC
                """,
                (since,),
            )
            overall = [dict(r) for r in cur.fetchall()]

    location_map: dict[str, list] = {}
    for row in rows:
        loc = row["location"]
        if loc not in location_map:
            location_map[loc] = []
        location_map[loc].append({"category": row["category"], "count": row["count"]})

    return {
        "period_days": days_back,
        "by_location": [
            {"location": loc, "top_categories": cats[:5]}
            for loc, cats in list(location_map.items())[:top_n]
        ],
        "overall_category_distribution": overall,
    }


def run_analytics_postgres() -> dict:
    """
    Query PostgreSQL directly (no BigQuery sync) and use Groq to generate
    a hotspot report. Used as the live fallback in GET /analytics/heatmap.
    """
    route_stats = _fetch_route_stats_postgres(90)
    temporal_stats = _fetch_temporal_patterns_postgres(90)
    category_stats = _fetch_category_hotspots_postgres(90, top_n=10)
    return generate_hotspot_report(route_stats, temporal_stats, category_stats)


@app.get("/analytics/hotspots")
def get_hotspots(top: int = 10):
    """
    Return the top N high-risk routes/stations from the latest stored report.
    Falls back to a live PostgreSQL aggregation if no report exists.
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
        logger.warning(f"Could not query hotspot_reports (migration pending?): {e}")

    if row is None:
        try:
            live = _fetch_route_stats_postgres(30)
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
            "note": "Run POST /analytics/run to generate a full report",
            "hotspots": locations,
        }

    hotspots = (row["hotspots"] or [])[:top]
    return {
        "source": "stored_report",
        "report_date": row["report_date"].isoformat() if row["report_date"] else None,
        "summary": row["ai_summary"],
        "hotspots": hotspots,
    }


@app.get("/analytics/temporal")
def get_temporal():
    """
    Return:
    - by_day_of_week: avg hour per day (0 for days with no data) for the continuous line
    - reports: each individual lost_report as {day_num, hour} for scatter dots
    """
    DAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"]

    # date_lost is now TIMESTAMPTZ — use it for both day-of-week and hour.
    # Fall back to created_at if date_lost is null.
    try:
        with get_conn() as conn:
            with conn.cursor() as cur:
                ts_expr = "COALESCE(date_lost, created_at)"
                cur.execute(
                    f"""
                    SELECT
                        EXTRACT(DOW FROM {ts_expr})::int AS day_num,
                        AVG(EXTRACT(HOUR FROM {ts_expr})
                            + EXTRACT(MINUTE FROM {ts_expr}) / 60.0) AS avg_hour,
                        COUNT(*) AS report_count
                    FROM lost_reports
                    WHERE {ts_expr} IS NOT NULL
                    GROUP BY day_num
                    ORDER BY day_num
                    """
                )
                agg_rows = cur.fetchall()

                cur.execute(
                    f"""
                    SELECT
                        EXTRACT(DOW FROM {ts_expr})::int AS day_num,
                        EXTRACT(HOUR FROM {ts_expr})
                            + EXTRACT(MINUTE FROM {ts_expr}) / 60.0 AS hour
                    FROM lost_reports
                    WHERE {ts_expr} IS NOT NULL
                    ORDER BY day_num, hour
                    """
                )
                detail_rows = cur.fetchall()
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Temporal query failed: {e}")

    day_map = {
        int(r["day_num"]): round(float(r["avg_hour"]), 3)
        for r in agg_rows
        if r["avg_hour"] is not None
    }

    # All 7 days; days with no data get avg_hour=None (frontend uses 0 for continuity)
    by_day = [
        {
            "day": DAYS[i],
            "day_num": i,
            "avg_hour": day_map.get(i),
            "count": next(
                (int(r["report_count"]) for r in agg_rows if int(r["day_num"]) == i), 0
            ),
        }
        for i in range(7)
    ]

    reports = [
        {"day_num": int(r["day_num"]), "hour": round(float(r["hour"]), 3)}
        for r in detail_rows
    ]

    return {"by_day_of_week": by_day, "reports": reports}


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
