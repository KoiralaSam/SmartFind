from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class ExtractRequest(_message.Message):
    __slots__ = ("image_base64",)
    IMAGE_BASE64_FIELD_NUMBER: _ClassVar[int]
    image_base64: str
    def __init__(self, image_base64: _Optional[str] = ...) -> None: ...

class ExtractResponse(_message.Message):
    __slots__ = ("item_name", "item_type", "category", "brand", "model", "color", "material", "item_condition", "item_description")
    ITEM_NAME_FIELD_NUMBER: _ClassVar[int]
    ITEM_TYPE_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    BRAND_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    MATERIAL_FIELD_NUMBER: _ClassVar[int]
    ITEM_CONDITION_FIELD_NUMBER: _ClassVar[int]
    ITEM_DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    item_name: str
    item_type: str
    category: str
    brand: str
    model: str
    color: str
    material: str
    item_condition: str
    item_description: str
    def __init__(self, item_name: _Optional[str] = ..., item_type: _Optional[str] = ..., category: _Optional[str] = ..., brand: _Optional[str] = ..., model: _Optional[str] = ..., color: _Optional[str] = ..., material: _Optional[str] = ..., item_condition: _Optional[str] = ..., item_description: _Optional[str] = ...) -> None: ...
