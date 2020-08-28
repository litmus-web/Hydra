import asyncio

from io import StringIO
from concurrent.futures import ThreadPoolExecutor
from typing import Coroutine, Any

import typing

from .response import OutGoingResponse


class ServerInfo:
    def __init__(self, port: int = None):
        self.port = port


def _map_new_header(header: tuple):
    return "HTTP_%s" % header[0], header[1]


def _format_headers(headers: dict):
    return dict(map(_map_new_header, headers.items()))


def _to_environ(msg: dict, server_info: ServerInfo):
    return {
        "REQUEST_METHOD": msg["method"],
        "PATH_INFO": msg["path"],
        "SERVER_PROTOCOL": msg["version"],
        "SERVER_NAME": "Sandman",
        "SERVER_PORT": str(server_info.port),

        "wsgi.input": StringIO(msg["body"]),
        "wsgi.url_scheme": msg["path"],
        **_format_headers(msg["headers"])
    }


def _handle_sync(req_id: int, app: typing.Callable, msg: dict, server_info: ServerInfo):
    out = OutGoingResponse(req_id)
    environ_dict = _to_environ(msg, server_info)
    response_body = app(environ_dict, WSGICallable(out))
    for body in response_body:
        out.body += body.decode('utf-8')
    return out


class WSGICallable:
    def __init__(self, out_response: OutGoingResponse):
        self._out_going = out_response

    def __call__(self, status, headers):
        self._out_going.status = int(status.split(" ")[0])
        self._out_going.headers = headers


class WSGIAdapter:
    def __init__(self):
        self._thread_pool = ThreadPoolExecutor()
        self.server_info = ServerInfo()

    def __call__(self, app, msg: dict) -> Coroutine[Any, Any, OutGoingResponse]:
        return self._handle_incoming(msg["request_id"], app, msg)

    async def _handle_incoming(self, req_id: int, app: typing.Callable, msg: dict) -> OutGoingResponse:
        return await asyncio.get_event_loop().run_in_executor(
            self._thread_pool,
            _handle_sync,
            req_id,
            app,
            msg,
            self.server_info,
        )
