import json
import os
import re
import sys
from dataclasses import dataclass
from datetime import datetime, timezone
from difflib import SequenceMatcher
from pathlib import Path
from typing import Any, Dict, Optional, Tuple
from zoneinfo import ZoneInfo

import grpc
from google.protobuf.json_format import MessageToDict
from google.protobuf.timestamp_pb2 import Timestamp

def _find_proto_py_passenger_dir() -> Path:
    env_override = (os.environ.get("PROTO_PY_PASSENGER_DIR") or "").strip()
    if env_override:
        p = Path(env_override).expanduser().resolve()
        if (p / "passenger_pb2.py").exists() and (p / "passenger_pb2_grpc.py").exists():
            return p

    here = Path(__file__).resolve()
    candidates = [
        here.parent / "shared" / "proto_py" / "passenger",
        Path.cwd() / "shared" / "proto_py" / "passenger",
    ]

    for base in here.parents:
        candidates.append(base / "shared" / "proto_py" / "passenger")

    for c in candidates:
        if (c / "passenger_pb2.py").exists() and (c / "passenger_pb2_grpc.py").exists():
            return c.resolve()

    raise RuntimeError(
        "Could not locate generated gRPC python files. "
        "Expected shared/proto_py/passenger/passenger_pb2.py to exist. "
        "Set PROTO_PY_PASSENGER_DIR to the folder containing passenger_pb2.py."
    )


def _find_proto_py_staff_dir() -> Path:
    env_override = (os.environ.get("PROTO_PY_STAFF_DIR") or "").strip()
    if env_override:
        p = Path(env_override).expanduser().resolve()
        if (p / "staff_pb2.py").exists() and (p / "staff_pb2_grpc.py").exists():
            return p

    here = Path(__file__).resolve()
    candidates = [
        here.parent / "shared" / "proto_py" / "staff",
        Path.cwd() / "shared" / "proto_py" / "staff",
    ]

    for base in here.parents:
        candidates.append(base / "shared" / "proto_py" / "staff")

    for c in candidates:
        if (c / "staff_pb2.py").exists() and (c / "staff_pb2_grpc.py").exists():
            return c.resolve()

    raise RuntimeError(
        "Could not locate generated gRPC python files. "
        "Expected shared/proto_py/staff/staff_pb2.py to exist. "
        "Set PROTO_PY_STAFF_DIR to the folder containing staff_pb2.py."
    )


_PROTO_PY_PASSENGER_DIR = _find_proto_py_passenger_dir()
_PROTO_PY_STAFF_DIR = _find_proto_py_staff_dir()

# The generated modules use absolute imports like `import passenger_pb2`,
# so we must ensure the generated folder is on sys.path.
if str(_PROTO_PY_PASSENGER_DIR) not in sys.path:
    sys.path.insert(0, str(_PROTO_PY_PASSENGER_DIR))
if str(_PROTO_PY_STAFF_DIR) not in sys.path:
    sys.path.insert(0, str(_PROTO_PY_STAFF_DIR))

_CENTRAL_TZ = ZoneInfo("America/Chicago")

import passenger_pb2  # noqa: E402
import passenger_pb2_grpc  # noqa: E402
import staff_pb2  # noqa: E402
import staff_pb2_grpc  # noqa: E402


DEFAULT_PASSENGER_SERVICE_ADDRESS = "passenger-service:50051"
DEFAULT_STAFF_SERVICE_ADDRESS = "staff-service:50052"


class ChatAction:
    CREATE_LOST_REPORT = "create_lost_report"
    CHECK_MY_LOST_ITEM = "check_my_lost_item"
    LIST_LOST_REPORTS = "list_lost_reports"
    LIST_MY_CLAIMS = "list_my_claims"
    DELETE_LOST_REPORT = "delete_lost_report"
    SEARCH_FOUND_ITEM_MATCHES = "search_found_item_matches"
    FILE_CLAIM = "file_claim"
    NONE = "none"


@dataclass(frozen=True)
class ChatDispatchResult:
    action: str
    ok: bool
    data: Dict[str, Any]
    error: Optional[str] = None


def _normalize_route_text(text: str) -> str:
    cleaned = re.sub(r"[^a-z0-9]+", " ", (text or "").strip().lower())
    return " ".join(cleaned.split())


def _split_route_name(route_name: str) -> tuple[str, str]:
    raw = (route_name or "").strip()
    if not raw:
        return "", ""

    parts = re.split(r"\s*(?:->|→|–|—|\bto\b|-)\s*", raw, maxsplit=1, flags=re.IGNORECASE)
    if len(parts) == 2:
        return _normalize_route_text(parts[0]), _normalize_route_text(parts[1])

    return _normalize_route_text(raw), ""


def _text_similarity(left: str, right: str) -> float:
    if not left or not right:
        return 0.0
    if left == right:
        return 1.0
    if left in right or right in left:
        return 0.92
    return SequenceMatcher(None, left, right).ratio()


def _score_route_match(route_name: str, route_from: str, route_to: str) -> float:
    route_norm = _normalize_route_text(route_name)
    from_norm = _normalize_route_text(route_from)
    to_norm = _normalize_route_text(route_to)
    if not route_norm or (not from_norm and not to_norm):
        return 0.0

    route_start, route_end = _split_route_name(route_name)
    query_norm = " ".join(part for part in (from_norm, to_norm) if part)
    score = 0.0

    # Prefer matching the actual route endpoints when the route_name contains them.
    if route_start:
        start_similarity = _text_similarity(from_norm, route_start) if from_norm else 0.0
        score += 0.70 * start_similarity
        if from_norm and from_norm == route_start:
            score += 0.45
    if route_end:
        end_similarity = _text_similarity(to_norm, route_end) if to_norm else 0.0
        score += 0.70 * end_similarity
        if to_norm and to_norm == route_end:
            score += 0.45

    if route_start and route_end and from_norm and to_norm:
        ordered_exact = from_norm == route_start and to_norm == route_end
        reversed_exact = from_norm == route_end and to_norm == route_start
        if ordered_exact:
            score += 1.20
        elif reversed_exact:
            score += 0.20
        else:
            ordered_similarity = (
                _text_similarity(from_norm, route_start) + _text_similarity(to_norm, route_end)
            ) / 2.0
            reversed_similarity = (
                _text_similarity(from_norm, route_end) + _text_similarity(to_norm, route_start)
            ) / 2.0
            score += 0.45 * ordered_similarity
            score += 0.10 * reversed_similarity

    if query_norm:
        score += 0.40 * SequenceMatcher(None, query_norm, route_norm).ratio()
        score += 0.15 * SequenceMatcher(
            None,
            f"{from_norm} to {to_norm}".strip(),
            route_norm,
        ).ratio()

    if from_norm:
        if from_norm in route_norm:
            score += 0.25
        else:
            score += 0.10 * SequenceMatcher(None, from_norm, route_norm).ratio()

    if to_norm:
        if to_norm in route_norm:
            score += 0.25
        else:
            score += 0.10 * SequenceMatcher(None, to_norm, route_norm).ratio()

    if from_norm and to_norm:
        if from_norm in route_norm and to_norm in route_norm:
            score += 0.35
        elif from_norm in route_norm or to_norm in route_norm:
            score += 0.10

    return score


def _parse_json_object(text: str) -> Optional[Dict[str, Any]]:
    t = (text or "").strip()
    if not (t.startswith("{") and t.endswith("}")):
        return None
    try:
        obj = json.loads(t)
    except json.JSONDecodeError:
        return None
    if isinstance(obj, dict):
        return obj
    return None


def _resolve_relative_date(text: str) -> Optional[datetime.date.__class__]:
    """Handle natural-language relative dates the LLM commonly produces."""
    from datetime import date, timedelta

    lower = text.strip().lower()
    today = datetime.now(_CENTRAL_TZ).date()
    relative_map = {
        "today": today,
        "yesterday": today - timedelta(days=1),
        "day before yesterday": today - timedelta(days=2),
    }
    if lower in relative_map:
        return relative_map[lower]
    return None


def _parse_date_time(date_str: str, time_str: str) -> datetime:
    ds = (date_str or "").strip()
    ts = (time_str or "").strip()

    if not ds or ds.lower() == "unknown":
        raise ValueError("date_lost is required (got empty/unknown)")
    if not ts or ts.lower() == "unknown":
        raise ValueError("time_lost is required (got empty/unknown)")

    date_formats = ("%Y-%m-%d", "%m/%d/%Y", "%d/%m/%Y", "%B %d, %Y", "%b %d, %Y")
    time_formats = ("%H:%M", "%H:%M:%S", "%I:%M %p", "%I:%M%p", "%I %p", "%I%p")

    parsed_date = _resolve_relative_date(ds)
    if parsed_date is None:
        for fmt in date_formats:
            try:
                parsed_date = datetime.strptime(ds, fmt).date()
                break
            except ValueError:
                continue
    if parsed_date is None:
        try:
            parsed_date = datetime.fromisoformat(ds).date()
        except ValueError:
            raise ValueError(f"Could not parse date_lost: '{ds}'")

    parsed_time = None
    normalized_ts = ts.upper().replace(".", "").strip()
    for fmt in time_formats:
        try:
            parsed_time = datetime.strptime(normalized_ts, fmt).time()
            break
        except ValueError:
            continue
    if parsed_time is None:
        try:
            parsed_time = datetime.fromisoformat(ts).time()
        except ValueError:
            raise ValueError(f"Could not parse time_lost: '{ts}'")

    # Interpret date/time in Central time, then normalize to UTC.
    combined = datetime.combine(parsed_date, parsed_time)
    local_dt = combined.replace(tzinfo=_CENTRAL_TZ)
    return local_dt.astimezone(timezone.utc)


def _to_timestamp(dt: datetime) -> Timestamp:
    ts = Timestamp()
    ts.FromDatetime(dt)
    return ts


def _intake_to_create_lost_report_request(
    *, passenger_id: str, intake: Dict[str, Any]
) -> passenger_pb2.CreateLostReportRequest:
    item_name = (intake.get("item_name") or "").strip()
    if not item_name or item_name.lower() == "unknown":
        raise ValueError("item_name is required (got empty/unknown)")

    color = (intake.get("color") or "unknown").strip() or "unknown"
    brand = (intake.get("brand") or "unknown").strip() or "unknown"
    description = (intake.get("description") or "unknown").strip() or "unknown"
    route_from = (intake.get("route_from") or "unknown").strip() or "unknown"
    route_to = (intake.get("route_to") or "unknown").strip() or "unknown"
    date_lost = (intake.get("date_lost") or "").strip()
    time_lost = (intake.get("time_lost") or "").strip()

    dt = _parse_date_time(date_lost, time_lost)

    return passenger_pb2.CreateLostReportRequest(
        passenger_id=passenger_id,
        item_name=item_name,
        item_description=description,
        item_type=(intake.get("item_type") or "unknown").strip() or "unknown",
        brand=brand,
        model=(intake.get("model") or "unknown").strip() or "unknown",
        color=color,
        material=(intake.get("material") or "unknown").strip() or "unknown",
        item_condition=(intake.get("item_condition") or "unknown").strip() or "unknown",
        category=(intake.get("category") or "unknown").strip() or "unknown",
        location_lost=(intake.get("location_lost") or "unknown").strip() or "unknown",
        route_or_station=(intake.get("route_or_station") or f"{route_from} -> {route_to}").strip()
        or "unknown",
        route_id=(intake.get("route_id") or "").strip(),
        date_lost=_to_timestamp(dt),
    )


def _infer_action(payload: Dict[str, Any]) -> str:
    action = (payload.get("action") or payload.get("intent") or "").strip().lower()
    normalized = action.replace("-", "_")
    if normalized in (
        "createlostreport",
        "create_lost_report",
        "create_lost_report_request",
    ):
        return ChatAction.CREATE_LOST_REPORT
    if normalized in ("checkmylostitem", "check_my_lost_item", "check_lost_item", "check_status"):
        return ChatAction.CHECK_MY_LOST_ITEM
    if normalized in ("listlostreports", "list_lost_reports"):
        return ChatAction.LIST_LOST_REPORTS
    if normalized in ("listclaims", "list_claims", "list_my_claims", "show_claims", "my_claims"):
        return ChatAction.LIST_MY_CLAIMS
    if normalized in ("deletelostreport", "delete_lost_report"):
        return ChatAction.DELETE_LOST_REPORT
    if normalized in ("searchfounditemmatches", "search_found_item_matches"):
        return ChatAction.SEARCH_FOUND_ITEM_MATCHES
    if normalized in ("fileclaim", "file_claim"):
        return ChatAction.FILE_CLAIM

    # Back-compat: current chat flow outputs the intake JSON directly.
    intake_keys = {
        "item_name",
        "color",
        "brand",
        "description",
        "route_from",
        "route_to",
        "date_lost",
        "time_lost",
    }
    if any(k in payload for k in intake_keys):
        return ChatAction.CREATE_LOST_REPORT

    return ChatAction.NONE


class PassengerGrpcHandler:
    def __init__(
        self,
        *,
        address: Optional[str] = None,
        timeout_seconds: float = 10.0,
    ) -> None:
        self._address = (
            address
            or os.environ.get("PASSENGER_SERVICE_ADDRESS")
            or DEFAULT_PASSENGER_SERVICE_ADDRESS
        )
        self._staff_address = (
            os.environ.get("STAFF_SERVICE_ADDRESS") or DEFAULT_STAFF_SERVICE_ADDRESS
        )
        self._timeout_seconds = timeout_seconds
        self._channel: Optional[grpc.aio.Channel] = None
        self._stub: Optional[passenger_pb2_grpc.PassengerServiceStub] = None
        self._staff_channel: Optional[grpc.aio.Channel] = None
        self._staff_stub: Optional[staff_pb2_grpc.StaffServiceStub] = None

    async def _get_stub(self) -> passenger_pb2_grpc.PassengerServiceStub:
        if self._stub is not None:
            return self._stub
        self._channel = grpc.aio.insecure_channel(self._address)
        try:
            await self._channel.channel_ready()
        except Exception:
            await self._channel.close()
            self._channel = None
            raise
        self._stub = passenger_pb2_grpc.PassengerServiceStub(self._channel)
        return self._stub

    async def _get_staff_stub(self) -> staff_pb2_grpc.StaffServiceStub:
        if self._staff_stub is not None:
            return self._staff_stub
        self._staff_channel = grpc.aio.insecure_channel(self._staff_address)
        try:
            await self._staff_channel.channel_ready()
        except Exception:
            await self._staff_channel.close()
            self._staff_channel = None
            raise
        self._staff_stub = staff_pb2_grpc.StaffServiceStub(self._staff_channel)
        return self._staff_stub

    def _metadata(self, forwarded_token: Optional[str]) -> Tuple[Tuple[str, str], ...]:
        md: list[Tuple[str, str]] = []
        internal = (os.environ.get("INTERNAL_SERVICE_SECRET") or "").strip()
        if internal:
            md.append(("x-internal-token", internal))
        if forwarded_token:
            md.append(("x-forwarded-token", forwarded_token))
        return tuple(md)

    async def close(self) -> None:
        if self._channel is not None:
            await self._channel.close()
        if self._staff_channel is not None:
            await self._staff_channel.close()
        self._channel = None
        self._stub = None
        self._staff_channel = None
        self._staff_stub = None

    async def list_routes(self) -> list[Dict[str, Any]]:
        stub = await self._get_staff_stub()
        req = staff_pb2.ListRoutesRequest(limit=500, offset=0)
        resp = await stub.ListRoutes(
            req, timeout=self._timeout_seconds, metadata=self._metadata(None)
        )
        return MessageToDict(resp, preserving_proto_field_name=True).get("routes") or []

    async def _resolve_route_fields(self, intake: Dict[str, Any]) -> Dict[str, Any]:
        if not isinstance(intake, dict):
            return intake
        route_from = str(intake.get("route_from") or "").strip()
        route_to = str(intake.get("route_to") or "").strip()
        if not route_from and not route_to:
            return intake

        try:
            routes = await self.list_routes()
        except Exception:
            return intake

        best_route: Optional[Dict[str, Any]] = None
        best_score = 0.0
        second_best_score = 0.0
        for route in routes:
            score = _score_route_match(
                str(route.get("route_name") or ""),
                route_from,
                route_to,
            )
            if score > best_score:
                second_best_score = best_score
                best_score = score
                best_route = route
            elif score > second_best_score:
                second_best_score = score

        threshold = 1.15 if route_from and route_to else 0.70
        ambiguous_gap = 0.12 if route_from and route_to else 0.08
        if (
            not best_route
            or best_score < threshold
            or (second_best_score > 0 and (best_score - second_best_score) < ambiguous_gap)
        ):
            return intake

        resolved = dict(intake)
        resolved["route_id"] = str(best_route.get("id") or "").strip()
        resolved["route_or_station"] = (
            str(best_route.get("route_name") or "").strip()
            or f"{route_from} -> {route_to}".strip(" ->")
        )
        return resolved

    async def create_lost_report(
        self, *, passenger_id: str, payload: Dict[str, Any], forwarded_token: Optional[str] = None
    ) -> passenger_pb2.LostReport:
        stub = await self._get_stub()
        data = payload.get("data") if isinstance(payload.get("data"), dict) else payload
        data = await self._resolve_route_fields(dict(data or {}))
        req = _intake_to_create_lost_report_request(passenger_id=passenger_id, intake=data)
        return await stub.CreateLostReport(
            req, timeout=self._timeout_seconds, metadata=self._metadata(forwarded_token)
        )

    async def list_lost_reports(
        self, *, passenger_id: str, status: str = "", forwarded_token: Optional[str] = None
    ) -> passenger_pb2.ListLostReportsResponse:
        stub = await self._get_stub()
        req = passenger_pb2.ListLostReportsRequest(passenger_id=passenger_id, status=status)
        return await stub.ListLostReports(
            req, timeout=self._timeout_seconds, metadata=self._metadata(forwarded_token)
        )

    async def list_my_claims(
        self,
        *,
        passenger_id: str,
        status: str = "",
        limit: int = 50,
        offset: int = 0,
        forwarded_token: Optional[str] = None,
    ) -> passenger_pb2.ListMyClaimsResponse:
        stub = await self._get_stub()
        req = passenger_pb2.ListMyClaimsRequest(
            passenger_id=passenger_id,
            status=status,
            limit=int(limit),
            offset=int(offset),
        )
        return await stub.ListMyClaims(
            req, timeout=self._timeout_seconds, metadata=self._metadata(forwarded_token)
        )

    async def delete_lost_report(
        self, *, passenger_id: str, lost_report_id: str, forwarded_token: Optional[str] = None
    ) -> None:
        stub = await self._get_stub()
        req = passenger_pb2.DeleteLostReportRequest(
            passenger_id=passenger_id, lost_report_id=lost_report_id
        )
        await stub.DeleteLostReport(
            req, timeout=self._timeout_seconds, metadata=self._metadata(forwarded_token)
        )

    async def search_found_item_matches(
        self,
        *,
        passenger_id: str,
        lost_report_id: str,
        limit: int = 10,
        forwarded_token: Optional[str] = None,
    ) -> passenger_pb2.SearchFoundItemMatchesResponse:
        stub = await self._get_stub()
        req = passenger_pb2.SearchFoundItemMatchesRequest(
            passenger_id=passenger_id,
            lost_report_id=lost_report_id,
            limit=int(limit),
        )
        return await stub.SearchFoundItemMatches(
            req, timeout=self._timeout_seconds, metadata=self._metadata(forwarded_token)
        )

    async def check_my_lost_item(
        self,
        *,
        passenger_id: str,
        lost_report_id: str = "",
        status: str = "open",
        limit: int = 10,
        forwarded_token: Optional[str] = None,
        auto_default: bool = True,
    ) -> Dict[str, Any]:
        """Resolve the "check my item" flow.

        When ``lost_report_id`` is provided, use it directly.
        Otherwise list the passenger's reports and:
          - 0 reports  -> return empty result.
          - 1 report   -> use it.
          - >1 reports -> if ``auto_default`` is True, default to most recent
            (passenger-service orders by created_at DESC). If False, return
            ``needs_choice=True`` with the list so the caller can ask the
            passenger to pick one.
        """
        resp = await self.list_lost_reports(
            passenger_id=passenger_id, status=status, forwarded_token=forwarded_token
        )
        resp_dict = MessageToDict(resp, preserving_proto_field_name=True)
        reports = resp_dict.get("reports") or []

        chosen = None
        needs_choice = False
        if lost_report_id:
            for r in reports:
                if str(r.get("id") or "").strip() == lost_report_id:
                    chosen = r
                    break
        elif len(reports) == 1:
            chosen = reports[0]
        elif len(reports) > 1:
            if auto_default:
                chosen = reports[0]
            else:
                needs_choice = True

        matches: list[Dict[str, Any]] = []
        chosen_id = (chosen or {}).get("id") or ""
        if chosen_id:
            try:
                mresp = await self.search_found_item_matches(
                    passenger_id=passenger_id,
                    lost_report_id=chosen_id,
                    limit=limit,
                    forwarded_token=forwarded_token,
                )
                matches = (MessageToDict(mresp, preserving_proto_field_name=True).get("matches") or [])
            except Exception:
                matches = []

        return {
            "report": chosen or {},
            "lost_report_id": chosen_id,
            "matches": matches,
            "reports": reports,
            "needs_choice": needs_choice,
        }

    async def file_claim(
        self,
        *,
        passenger_id: str,
        found_item_id: str,
        lost_report_id: str,
        message: str,
        forwarded_token: Optional[str] = None,
    ) -> passenger_pb2.ItemClaim:
        stub = await self._get_stub()
        req = passenger_pb2.FileClaimRequest(
            passenger_id=passenger_id,
            found_item_id=found_item_id,
            lost_report_id=lost_report_id,
            message=message,
        )
        return await stub.FileClaim(
            req, timeout=self._timeout_seconds, metadata=self._metadata(forwarded_token)
        )

    async def dispatch_from_chat_reply(
        self, *, passenger_id: str, chat_reply_text: str, forwarded_token: Optional[str] = None
    ) -> ChatDispatchResult:
        payload = _parse_json_object(chat_reply_text)
        if payload is None:
            return ChatDispatchResult(action=ChatAction.NONE, ok=True, data={})

        action = _infer_action(payload)

        try:
            if action == ChatAction.CREATE_LOST_REPORT:
                report = await self.create_lost_report(
                    passenger_id=passenger_id, payload=payload, forwarded_token=forwarded_token
                )
                matches = []
                try:
                    resp = await self.search_found_item_matches(
                        passenger_id=passenger_id,
                        lost_report_id=report.id,
                        limit=10,
                        forwarded_token=forwarded_token,
                    )
                    resp_dict = MessageToDict(resp, preserving_proto_field_name=True)
                    matches = resp_dict.get("matches") or []
                except Exception:
                    matches = []
                return ChatDispatchResult(
                    action=action,
                    ok=True,
                    data={
                        "report": MessageToDict(report, preserving_proto_field_name=True),
                        "matches": matches,
                    },
                )

            if action == ChatAction.CHECK_MY_LOST_ITEM:
                data = payload.get("data") if isinstance(payload.get("data"), dict) else payload
                status = (data.get("status") or "open").strip() or "open"
                limit = int(data.get("limit") or 10)
                lost_report_id = (data.get("lost_report_id") or "").strip()
                # Default value preserves prior behavior; chat-agent may pass False to force disambiguation.
                auto_default_raw = data.get("auto_default")
                auto_default = True if auto_default_raw is None else bool(auto_default_raw)
                out = await self.check_my_lost_item(
                    passenger_id=passenger_id,
                    lost_report_id=lost_report_id,
                    status=status,
                    limit=limit,
                    forwarded_token=forwarded_token,
                    auto_default=auto_default,
                )
                return ChatDispatchResult(action=action, ok=True, data=out)

            if action == ChatAction.LIST_LOST_REPORTS:
                data = payload.get("data") if isinstance(payload.get("data"), dict) else payload
                status = (data.get("status") or "").strip()
                resp = await self.list_lost_reports(
                    passenger_id=passenger_id, status=status, forwarded_token=forwarded_token
                )
                return ChatDispatchResult(
                    action=action,
                    ok=True,
                    data=MessageToDict(resp, preserving_proto_field_name=True),
                )

            if action == ChatAction.LIST_MY_CLAIMS:
                data = payload.get("data") if isinstance(payload.get("data"), dict) else payload
                status = (data.get("status") or "").strip()
                limit = int(data.get("limit") or 50)
                offset = int(data.get("offset") or 0)
                resp = await self.list_my_claims(
                    passenger_id=passenger_id,
                    status=status,
                    limit=limit,
                    offset=offset,
                    forwarded_token=forwarded_token,
                )
                return ChatDispatchResult(
                    action=action,
                    ok=True,
                    data=MessageToDict(resp, preserving_proto_field_name=True),
                )

            if action == ChatAction.DELETE_LOST_REPORT:
                data = payload.get("data") if isinstance(payload.get("data"), dict) else payload
                lost_report_id = (data.get("lost_report_id") or data.get("id") or "").strip()
                if not lost_report_id:
                    raise ValueError("missing lost_report_id for delete_lost_report")
                await self.delete_lost_report(
                    passenger_id=passenger_id,
                    lost_report_id=lost_report_id,
                    forwarded_token=forwarded_token,
                )
                return ChatDispatchResult(action=action, ok=True, data={})

            if action == ChatAction.SEARCH_FOUND_ITEM_MATCHES:
                data = payload.get("data") if isinstance(payload.get("data"), dict) else payload
                lost_report_id = (data.get("lost_report_id") or "").strip()
                if not lost_report_id:
                    out = await self.check_my_lost_item(
                        passenger_id=passenger_id,
                        lost_report_id="",
                        status="open",
                        limit=int(data.get("limit") or 10),
                        forwarded_token=forwarded_token,
                    )
                    return ChatDispatchResult(action=ChatAction.CHECK_MY_LOST_ITEM, ok=True, data=out)
                limit = int(data.get("limit") or 10)
                resp = await self.search_found_item_matches(
                    passenger_id=passenger_id,
                    lost_report_id=lost_report_id,
                    limit=limit,
                    forwarded_token=forwarded_token,
                )
                return ChatDispatchResult(
                    action=action,
                    ok=True,
                    data=MessageToDict(resp, preserving_proto_field_name=True),
                )

            if action == ChatAction.FILE_CLAIM:
                data = payload.get("data") if isinstance(payload.get("data"), dict) else payload
                found_item_id = (data.get("found_item_id") or "").strip()
                lost_report_id = (data.get("lost_report_id") or "").strip()
                message = (data.get("message") or "").strip()
                if not found_item_id or not lost_report_id:
                    raise ValueError("missing found_item_id or lost_report_id for file_claim")
                claim = await self.file_claim(
                    passenger_id=passenger_id,
                    found_item_id=found_item_id,
                    lost_report_id=lost_report_id,
                    message=message,
                    forwarded_token=forwarded_token,
                )
                return ChatDispatchResult(
                    action=action,
                    ok=True,
                    data=MessageToDict(claim, preserving_proto_field_name=True),
                )

            return ChatDispatchResult(action=ChatAction.NONE, ok=True, data={})

        except grpc.aio.AioRpcError as e:
            return ChatDispatchResult(
                action=action,
                ok=False,
                data={},
                error=f"grpc error: code={e.code().name} message={e.details()}",
            )
        except Exception as e:
            return ChatDispatchResult(action=action, ok=False, data={}, error=str(e))
