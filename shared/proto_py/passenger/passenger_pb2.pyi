import datetime

from google.protobuf import empty_pb2 as _empty_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Passenger(_message.Message):
    __slots__ = ("id", "email", "full_name", "phone", "created_at", "updated_at", "avatar_url")
    ID_FIELD_NUMBER: _ClassVar[int]
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    FULL_NAME_FIELD_NUMBER: _ClassVar[int]
    PHONE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    AVATAR_URL_FIELD_NUMBER: _ClassVar[int]
    id: str
    email: str
    full_name: str
    phone: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    avatar_url: str
    def __init__(self, id: _Optional[str] = ..., email: _Optional[str] = ..., full_name: _Optional[str] = ..., phone: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., avatar_url: _Optional[str] = ...) -> None: ...

class LoginRequest(_message.Message):
    __slots__ = ("id_token",)
    ID_TOKEN_FIELD_NUMBER: _ClassVar[int]
    id_token: str
    def __init__(self, id_token: _Optional[str] = ...) -> None: ...

class LoginResponse(_message.Message):
    __slots__ = ("passenger", "session_token")
    PASSENGER_FIELD_NUMBER: _ClassVar[int]
    SESSION_TOKEN_FIELD_NUMBER: _ClassVar[int]
    passenger: Passenger
    session_token: str
    def __init__(self, passenger: _Optional[_Union[Passenger, _Mapping]] = ..., session_token: _Optional[str] = ...) -> None: ...

class LostReport(_message.Message):
    __slots__ = ("id", "reporter_passenger_id", "item_name", "item_description", "item_type", "brand", "model", "color", "material", "item_condition", "category", "location_lost", "route_or_station", "route_id", "date_lost", "status", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    REPORTER_PASSENGER_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_NAME_FIELD_NUMBER: _ClassVar[int]
    ITEM_DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ITEM_TYPE_FIELD_NUMBER: _ClassVar[int]
    BRAND_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    MATERIAL_FIELD_NUMBER: _ClassVar[int]
    ITEM_CONDITION_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    LOCATION_LOST_FIELD_NUMBER: _ClassVar[int]
    ROUTE_OR_STATION_FIELD_NUMBER: _ClassVar[int]
    ROUTE_ID_FIELD_NUMBER: _ClassVar[int]
    DATE_LOST_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    reporter_passenger_id: str
    item_name: str
    item_description: str
    item_type: str
    brand: str
    model: str
    color: str
    material: str
    item_condition: str
    category: str
    location_lost: str
    route_or_station: str
    route_id: str
    date_lost: _timestamp_pb2.Timestamp
    status: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., reporter_passenger_id: _Optional[str] = ..., item_name: _Optional[str] = ..., item_description: _Optional[str] = ..., item_type: _Optional[str] = ..., brand: _Optional[str] = ..., model: _Optional[str] = ..., color: _Optional[str] = ..., material: _Optional[str] = ..., item_condition: _Optional[str] = ..., category: _Optional[str] = ..., location_lost: _Optional[str] = ..., route_or_station: _Optional[str] = ..., route_id: _Optional[str] = ..., date_lost: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., status: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CreateLostReportRequest(_message.Message):
    __slots__ = ("passenger_id", "item_name", "item_description", "item_type", "brand", "model", "color", "material", "item_condition", "category", "location_lost", "route_or_station", "route_id", "date_lost")
    PASSENGER_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_NAME_FIELD_NUMBER: _ClassVar[int]
    ITEM_DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ITEM_TYPE_FIELD_NUMBER: _ClassVar[int]
    BRAND_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    MATERIAL_FIELD_NUMBER: _ClassVar[int]
    ITEM_CONDITION_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    LOCATION_LOST_FIELD_NUMBER: _ClassVar[int]
    ROUTE_OR_STATION_FIELD_NUMBER: _ClassVar[int]
    ROUTE_ID_FIELD_NUMBER: _ClassVar[int]
    DATE_LOST_FIELD_NUMBER: _ClassVar[int]
    passenger_id: str
    item_name: str
    item_description: str
    item_type: str
    brand: str
    model: str
    color: str
    material: str
    item_condition: str
    category: str
    location_lost: str
    route_or_station: str
    route_id: str
    date_lost: _timestamp_pb2.Timestamp
    def __init__(self, passenger_id: _Optional[str] = ..., item_name: _Optional[str] = ..., item_description: _Optional[str] = ..., item_type: _Optional[str] = ..., brand: _Optional[str] = ..., model: _Optional[str] = ..., color: _Optional[str] = ..., material: _Optional[str] = ..., item_condition: _Optional[str] = ..., category: _Optional[str] = ..., location_lost: _Optional[str] = ..., route_or_station: _Optional[str] = ..., route_id: _Optional[str] = ..., date_lost: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListLostReportsRequest(_message.Message):
    __slots__ = ("passenger_id", "status")
    PASSENGER_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    passenger_id: str
    status: str
    def __init__(self, passenger_id: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class ListLostReportsResponse(_message.Message):
    __slots__ = ("reports",)
    REPORTS_FIELD_NUMBER: _ClassVar[int]
    reports: _containers.RepeatedCompositeFieldContainer[LostReport]
    def __init__(self, reports: _Optional[_Iterable[_Union[LostReport, _Mapping]]] = ...) -> None: ...

class DeleteLostReportRequest(_message.Message):
    __slots__ = ("passenger_id", "lost_report_id")
    PASSENGER_ID_FIELD_NUMBER: _ClassVar[int]
    LOST_REPORT_ID_FIELD_NUMBER: _ClassVar[int]
    passenger_id: str
    lost_report_id: str
    def __init__(self, passenger_id: _Optional[str] = ..., lost_report_id: _Optional[str] = ...) -> None: ...

class FoundItemMatch(_message.Message):
    __slots__ = ("found_item_id", "item_name", "item_description", "item_type", "brand", "model", "color", "material", "item_condition", "category", "location_found", "route_or_station", "route_id", "date_found", "status", "similarity_score")
    FOUND_ITEM_ID_FIELD_NUMBER: _ClassVar[int]
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
    SIMILARITY_SCORE_FIELD_NUMBER: _ClassVar[int]
    found_item_id: str
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
    similarity_score: float
    def __init__(self, found_item_id: _Optional[str] = ..., item_name: _Optional[str] = ..., item_description: _Optional[str] = ..., item_type: _Optional[str] = ..., brand: _Optional[str] = ..., model: _Optional[str] = ..., color: _Optional[str] = ..., material: _Optional[str] = ..., item_condition: _Optional[str] = ..., category: _Optional[str] = ..., location_found: _Optional[str] = ..., route_or_station: _Optional[str] = ..., route_id: _Optional[str] = ..., date_found: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., status: _Optional[str] = ..., similarity_score: _Optional[float] = ...) -> None: ...

class SearchFoundItemMatchesRequest(_message.Message):
    __slots__ = ("passenger_id", "lost_report_id", "limit")
    PASSENGER_ID_FIELD_NUMBER: _ClassVar[int]
    LOST_REPORT_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    passenger_id: str
    lost_report_id: str
    limit: int
    def __init__(self, passenger_id: _Optional[str] = ..., lost_report_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class SearchFoundItemMatchesResponse(_message.Message):
    __slots__ = ("matches",)
    MATCHES_FIELD_NUMBER: _ClassVar[int]
    matches: _containers.RepeatedCompositeFieldContainer[FoundItemMatch]
    def __init__(self, matches: _Optional[_Iterable[_Union[FoundItemMatch, _Mapping]]] = ...) -> None: ...

class FileClaimRequest(_message.Message):
    __slots__ = ("passenger_id", "found_item_id", "lost_report_id", "message")
    PASSENGER_ID_FIELD_NUMBER: _ClassVar[int]
    FOUND_ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    LOST_REPORT_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    passenger_id: str
    found_item_id: str
    lost_report_id: str
    message: str
    def __init__(self, passenger_id: _Optional[str] = ..., found_item_id: _Optional[str] = ..., lost_report_id: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

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
