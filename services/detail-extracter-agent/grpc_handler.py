import sys
from pathlib import Path
import os
from typing import Optional
from concurrent import futures

import grpc

from extractor import GOOGLE_SERVICE_ACCOUNT_JSON, logger, run_extraction


def _find_proto_py_detailextractor_dir() -> Path:
    env_override = (os.environ.get("PROTO_PY_DETAILEXTRACTOR_DIR") or "").strip()
    if env_override:
        p = Path(env_override).expanduser().resolve()
        if (p / "detailextractor_pb2.py").exists() and (p / "detailextractor_pb2_grpc.py").exists():
            return p

    here = Path(__file__).resolve()
    candidates = [
        here.parent / "shared" / "proto_py" / "detailextractor",
        Path.cwd() / "shared" / "proto_py" / "detailextractor",
    ]

    for base in here.parents:
        candidates.append(base / "shared" / "proto_py" / "detailextractor")

    for c in candidates:
        if (c / "detailextractor_pb2.py").exists() and (c / "detailextractor_pb2_grpc.py").exists():
            return c.resolve()

    raise RuntimeError(
        "Could not locate generated gRPC python files. "
        "Expected shared/proto_py/detailextractor/detailextractor_pb2.py to exist. "
        "Set PROTO_PY_DETAILEXTRACTOR_DIR to the folder containing detailextractor_pb2.py."
    )


_PROTO_PY_DIR = _find_proto_py_detailextractor_dir()
if str(_PROTO_PY_DIR) not in sys.path:
    sys.path.insert(0, str(_PROTO_PY_DIR))

import detailextractor_pb2  # noqa: E402
import detailextractor_pb2_grpc  # noqa: E402

DEFAULT_GRPC_ADDR = os.environ.get("DETAIL_EXTRACTER_GRPC_ADDR", "0.0.0.0:50053")

_server: Optional[grpc.Server] = None


def _metadata_dict(context: grpc.ServicerContext) -> dict[str, str]:
    md: dict[str, str] = {}
    for k, v in context.invocation_metadata():
        if k:
            md[k.lower()] = v
    return md


class DetailExtractorGRPCServicer(detailextractor_pb2_grpc.DetailExtractorServiceServicer):
    def Extract(  # noqa: N802 (protoc naming)
        self, request: detailextractor_pb2.ExtractRequest, context: grpc.ServicerContext
    ) -> detailextractor_pb2.ExtractResponse:
        md = _metadata_dict(context)

        expected_internal = (os.environ.get("INTERNAL_SERVICE_SECRET") or "").strip()
        if not expected_internal:
            context.abort(grpc.StatusCode.FAILED_PRECONDITION, "INTERNAL_SERVICE_SECRET not configured")

        internal = (md.get("x-internal-token") or "").strip()
        if internal != expected_internal:
            context.abort(grpc.StatusCode.UNAUTHENTICATED, "missing or invalid internal token")

        forwarded = (md.get("x-forwarded-token") or "").strip()
        if not forwarded:
            context.abort(grpc.StatusCode.UNAUTHENTICATED, "missing forwarded token")

        image_b64 = (request.image_base64 or "").strip()
        if not image_b64:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "image_base64 is required")

        if not GOOGLE_SERVICE_ACCOUNT_JSON:
            context.abort(grpc.StatusCode.FAILED_PRECONDITION, "GOOGLE_SERVICE_ACCOUNT_JSON not configured")

        try:
            result = run_extraction(image_b64)
        except Exception as e:
            logger.exception("gRPC extraction failed")
            context.abort(grpc.StatusCode.INTERNAL, f"extraction failed: {e}")

        return detailextractor_pb2.ExtractResponse(**result)

def start_grpc_server(addr: str = DEFAULT_GRPC_ADDR) -> grpc.Server:
    global _server
    if _server is not None:
        return _server

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    detailextractor_pb2_grpc.add_DetailExtractorServiceServicer_to_server(
        DetailExtractorGRPCServicer(), server
    )
    server.add_insecure_port(addr)
    server.start()
    _server = server
    logger.info(f"Detail extractor gRPC server listening on {addr}")
    return server


def stop_grpc_server(grace_seconds: float = 5.0) -> None:
    global _server
    if _server is None:
        return
    _server.stop(grace_seconds)
    _server = None
