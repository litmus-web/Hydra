from typing import Coroutine, Any

from .response import OutGoingResponse
from ..codes import OpCodes

class ASGIAdapter:
    def __call__(self, msg: dict) -> Coroutine[Any, Any, OutGoingResponse]:
        return self._handle_incoming(msg["request_id"], msg)

    async def _handle_incoming(self, req_id: int, msg: dict) -> OutGoingResponse:
        return OutGoingResponse(
            req_id=req_id,
            status=200,
            headers=(),
            body="hello world",
        )