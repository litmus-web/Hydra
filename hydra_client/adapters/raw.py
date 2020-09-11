from typing import Coroutine, Any
from aiohttp import ClientWebSocketResponse

from ..helpers import dumps_data
from ..codes import OpCodes


class RawAdapter:
    def __call__(self, ws: ClientWebSocketResponse, app, msg: dict) -> Coroutine[Any, Any, None]:
        return self._handle_incoming(ws, msg["request_id"], msg)

    async def _handle_incoming(self, ws: ClientWebSocketResponse, req_id: int, msg: dict):
        first = {
            "op": OpCodes.HTTP_REQUEST,
            "meta_data": {
                "meta_response_type": "complete"
            },
            "request_id": req_id,
            "type": "response.start",
            "status": 200,
            "headers": (("hello", "world"),),
            "body": "hello world",
            "more_body": False
        }
        await ws.send_bytes(dumps_data(first))

        # second = {
        #    "op": OpCodes.HTTP_REQUEST,
        #    "request_id": req_id,
        #    "type": "response.body",
        #    "body": "hello world",
        #   "more_body": False
        # }
        # await ws.send_bytes(dumps_data(second))
