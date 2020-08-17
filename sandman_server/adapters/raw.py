from typing import Coroutine, Any
from aiohttp import ClientWebSocketResponse

from ..helpers import dumps_data
from .response import OutGoingResponse


class RawAdapter:
    def __call__(self, ws: ClientWebSocketResponse, app, msg: dict) -> Coroutine[Any, Any, None]:
        return self._handle_incoming(ws, msg["request_id"], msg)

    async def _handle_incoming(self, ws: ClientWebSocketResponse, req_id: int, msg: dict):
        first = {
            "request_id": req_id,
            "type": "response.start",
            "status": 200,
            "headers": ()
        }
        await ws.send_bytes(dumps_data(first))

        second = {
            "request_id": req_id,
            "type": "response.body",
            "body": "hello world",
            "more_body": False
        }
        await ws.send_bytes(dumps_data(second))
        