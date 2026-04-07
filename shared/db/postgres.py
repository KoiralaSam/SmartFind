"""
Postgres connection pool helpers for Python services.
"""

from __future__ import annotations

from psycopg_pool import ConnectionPool


pool: ConnectionPool | None = None


def init_db(db_url: str) -> None:
    global pool

    next_pool = ConnectionPool(conninfo=db_url, open=False)

    try:
        next_pool.open()
        with next_pool.connection():
            pass
    except Exception as exc:
        next_pool.close()
        raise RuntimeError("failed to connect to database") from exc

    pool = next_pool


def get_db() -> ConnectionPool:
    if pool is None:
        raise RuntimeError("database has not been initialized")

    return pool
