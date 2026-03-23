"""
Simple helpers to read environment variables with typed fallbacks.
"""

from __future__ import annotations

import os


def get_string(key: str, fallback: str) -> str:
    value = os.getenv(key)
    if value is None:
        return fallback
    return value


def get_int(key: str, fallback: int) -> int:
    value = os.getenv(key)
    if value is None:
        return fallback
    try:
        return int(value)
    except (TypeError, ValueError):
        return fallback


def get_bool(key: str, fallback: bool) -> bool:
    value = os.getenv(key)
    if value is None:
        return fallback

    normalized = value.strip().lower()
    if normalized in {"1", "true", "t", "yes", "y", "on"}:
        return True
    if normalized in {"0", "false", "f", "no", "n", "off"}:
        return False

    return fallback
