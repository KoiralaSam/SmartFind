import datetime

from google.protobuf import empty_pb2 as _empty_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Staff(_message.Message):
    __slots__ = ("id", "full_name", "email", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    FULL_NAME_FIELD_NUMBER: _ClassVar[int]
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    full_name: str
    email: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., full_name: _Optional[str] = ..., email: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class LoginRequest(_message.Message):
    __slots__ = ("email", "password")
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_FIELD_NUMBER: _ClassVar[int]
    email: str
    password: str
    def __init__(self, email: _Optional[str] = ..., password: _Optional[str] = ...) -> None: ...

class LoginResponse(_message.Message):
    __slots__ = ("staff", "session_token")
    STAFF_FIELD_NUMBER: _ClassVar[int]
    SESSION_TOKEN_FIELD_NUMBER: _ClassVar[int]
    staff: Staff
    session_token: str
    def __init__(self, staff: _Optional[_Union[Staff, _Mapping]] = ..., session_token: _Optional[str] = ...) -> None: ...

class CreateStaffRequest(_message.Message):
    __slots__ = ("transit_code", "full_name", "email", "password")
    TRANSIT_CODE_FIELD_NUMBER: _ClassVar[int]
    FULL_NAME_FIELD_NUMBER: _ClassVar[int]
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_FIELD_NUMBER: _ClassVar[int]
    transit_code: str
    full_name: str
    email: str
    password: str
    def __init__(self, transit_code: _Optional[str] = ..., full_name: _Optional[str] = ..., email: _Optional[str] = ..., password: _Optional[str] = ...) -> None: ...

class FoundItem(_message.Message):
    __slots__ = ("id", "posted_by_staff_id", "item_name", "item_description", "item_type", "brand", "model", "color", "material", "item_condition", "category", "location_found", "route_or_station", "route_id", "date_found", "status", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    POSTED_BY_STAFF_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_NAME_FIELD_NUMBER: _ClassVar[int]
    ITEM_DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ITEM_TYPE_FIELD_NUMBER: _ClassVar[int]
    BRAND_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    MATERIAL_FIELD_NUMBER: _ClassVar[int]
    ITEM_CONDITION_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FOUND_FIELD_NUMBER: _ClassVar[int]
    ROUTE_OR_STATION_FIELD_NUMBER: _ClassVar[int]
    ROUTE_ID_FIELD_NUMBER: _ClassVar[int]
    DATE_FOUND_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    posted_by_staff_id: str
    item_name: str
    item_description: str
    item_type: str
    brand: str
    model: str
    color: str
    material: str
    item_condition: str
    category: str
    location_found: str
    route_or_station: str
    route_id: str
    date_found: _timestamp_pb2.Timestamp
    status: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., posted_by_staff_id: _Optional[str] = ..., item_name: _Optional[str] = ..., item_description: _Optional[str] = ..., item_type: _Optional[str] = ..., brand: _Optional[str] = ..., model: _Optional[str] = ..., color: _Optional[str] = ..., material: _Optional[str] = ..., item_condition: _Optional[str] = ..., category: _Optional[str] = ..., location_found: _Optional[str] = ..., route_or_station: _Optional[str] = ..., route_id: _Optional[str] = ..., date_found: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., status: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CreateFoundItemRequest(_message.Message):
    __slots__ = ("staff_id", "item_name", "item_description", "item_type", "brand", "model", "color", "material", "item_condition", "category", "location_found", "route_or_station", "route_id", "date_found")
    STAFF_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_NAME_FIELD_NUMBER: _ClassVar[int]
    ITEM_DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ITEM_TYPE_FIELD_NUMBER: _ClassVar[int]
    BRAND_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    MATERIAL_FIELD_NUMBER: _ClassVar[int]
    ITEM_CONDITION_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FOUND_FIELD_NUMBER: _ClassVar[int]
    ROUTE_OR_STATION_FIELD_NUMBER: _ClassVar[int]
    ROUTE_ID_FIELD_NUMBER: _ClassVar[int]
    DATE_FOUND_FIELD_NUMBER: _ClassVar[int]
    staff_id: str
    item_name: str
    item_description: str
    item_type: str
    brand: str
    model: str
    color: str
    material: str
    item_condition: str
    category: str
    location_found: str
    route_or_station: str
    route_id: str
    date_found: _timestamp_pb2.Timestamp
    def __init__(self, staff_id: _Optional[str] = ..., item_name: _Optional[str] = ..., item_description: _Optional[str] = ..., item_type: _Optional[str] = ..., brand: _Optional[str] = ..., model: _Optional[str] = ..., color: _Optional[str] = ..., material: _Optional[str] = ..., item_condition: _Optional[str] = ..., category: _Optional[str] = ..., location_found: _Optional[str] = ..., route_or_station: _Optional[str] = ..., route_id: _Optional[str] = ..., date_found: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class UpdateFoundItemStatusRequest(_message.Message):
    __slots__ = ("staff_id", "found_item_id", "status")
    STAFF_ID_FIELD_NUMBER: _ClassVar[int]
    FOUND_ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    staff_id: str
    found_item_id: str
    status: str
    def __init__(self, staff_id: _Optional[str] = ..., found_item_id: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class ListFoundItemsRequest(_message.Message):
    __slots__ = ("status", "route_id", "posted_by_staff_id", "limit", "offset")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ROUTE_ID_FIELD_NUMBER: _ClassVar[int]
    POSTED_BY_STAFF_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    status: str
    route_id: str
    posted_by_staff_id: str
    limit: int
    offset: int
    def __init__(self, status: _Optional[str] = ..., route_id: _Optional[str] = ..., posted_by_staff_id: _Optional[str] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListFoundItemsResponse(_message.Message):
    __slots__ = ("items",)
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[FoundItem]
    def __init__(self, items: _Optional[_Iterable[_Union[FoundItem, _Mapping]]] = ...) -> None: ...

class ItemClaim(_message.Message):
    __slots__ = ("id", "item_id", "claimant_passenger_id", "lost_report_id", "message", "status", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    CLAIMANT_PASSENGER_ID_FIELD_NUMBER: _ClassVar[int]
    LOST_REPORT_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    item_id: str
    claimant_passenger_id: str
    lost_report_id: str
    message: str
    status: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., item_id: _Optional[str] = ..., claimant_passenger_id: _Optional[str] = ..., lost_report_id: _Optional[str] = ..., message: _Optional[str] = ..., status: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListClaimsRequest(_message.Message):
    __slots__ = ("status", "item_id", "passenger_id", "limit", "offset")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    PASSENGER_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    status: str
    item_id: str
    passenger_id: str
    limit: int
    offset: int
    def __init__(self, status: _Optional[str] = ..., item_id: _Optional[str] = ..., passenger_id: _Optional[str] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListClaimsResponse(_message.Message):
    __slots__ = ("claims",)
    CLAIMS_FIELD_NUMBER: _ClassVar[int]
    claims: _containers.RepeatedCompositeFieldContainer[ItemClaim]
    def __init__(self, claims: _Optional[_Iterable[_Union[ItemClaim, _Mapping]]] = ...) -> None: ...

class ReviewClaimRequest(_message.Message):
    __slots__ = ("staff_id", "claim_id", "decision")
    STAFF_ID_FIELD_NUMBER: _ClassVar[int]
    CLAIM_ID_FIELD_NUMBER: _ClassVar[int]
    DECISION_FIELD_NUMBER: _ClassVar[int]
    staff_id: str
    claim_id: str
    decision: str
    def __init__(self, staff_id: _Optional[str] = ..., claim_id: _Optional[str] = ..., decision: _Optional[str] = ...) -> None: ...

class Route(_message.Message):
    __slots__ = ("id", "route_name", "created_by_staff_id", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    ROUTE_NAME_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_STAFF_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    route_name: str
    created_by_staff_id: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., route_name: _Optional[str] = ..., created_by_staff_id: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CreateRouteRequest(_message.Message):
    __slots__ = ("staff_id", "route_name")
    STAFF_ID_FIELD_NUMBER: _ClassVar[int]
    ROUTE_NAME_FIELD_NUMBER: _ClassVar[int]
    staff_id: str
    route_name: str
    def __init__(self, staff_id: _Optional[str] = ..., route_name: _Optional[str] = ...) -> None: ...

class DeleteRouteRequest(_message.Message):
    __slots__ = ("staff_id", "route_id")
    STAFF_ID_FIELD_NUMBER: _ClassVar[int]
    ROUTE_ID_FIELD_NUMBER: _ClassVar[int]
    staff_id: str
    route_id: str
    def __init__(self, staff_id: _Optional[str] = ..., route_id: _Optional[str] = ...) -> None: ...

class ListRoutesRequest(_message.Message):
    __slots__ = ("created_by_staff_id", "limit", "offset")
    CREATED_BY_STAFF_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    created_by_staff_id: str
    limit: int
    offset: int
    def __init__(self, created_by_staff_id: _Optional[str] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListRoutesResponse(_message.Message):
    __slots__ = ("routes",)
    ROUTES_FIELD_NUMBER: _ClassVar[int]
    routes: _containers.RepeatedCompositeFieldContainer[Route]
    def __init__(self, routes: _Optional[_Iterable[_Union[Route, _Mapping]]] = ...) -> None: ...
