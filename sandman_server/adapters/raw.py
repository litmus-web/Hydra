import asyncio

from typing import Coroutine, Any
from aiohttp import ClientWebSocketResponse

from ..helpers import dumps_data


class RawAdapter:
    def __call__(self, ws: ClientWebSocketResponse, app, msg: dict) -> Coroutine[Any, Any, None]:
        return self._handle_incoming(ws, msg["request_id"], msg)

    async def _start_internal(self, ws: ClientWebSocketResponse, req_id: int, msg: dict):
        try:
            await asyncio.wait_for(self._handle_incoming(ws, req_id, msg), timeout=15)
        except asyncio.TimeoutError:
            first = {
                "request_id": req_id,
                "type": "response.timeout",
            }
            await ws.send_bytes(dumps_data(first))

    async def _handle_incoming(self, ws: ClientWebSocketResponse, req_id: int, msg: dict):
        first = {
            "meta_data": {
                "meta_response_type": "complete"
            },
            "request_id": req_id,
            "type": "response.start",
            "status": 200,
            "headers": (),
            "body": "hello world",
            "more_body": False
        }
        await ws.send_bytes(dumps_data(first))

        #second = {
        #    "request_id": req_id,
        #    "type": "response.body",
        #    "body": "hello world",
         #   "more_body": False
        #}
        #await ws.send_bytes(dumps_data(second))
        