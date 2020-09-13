import typing
import importlib
import logging

from aiohttp import ClientWebSocketResponse

from .websocket import AutoShardedWorker
from ..adapters.asgi import ASGIAdapter
from ..adapters.wsgi import WSGIAdapter
from ..adapters.raw import RawAdapter

logger = logging.getLogger("Hydra-Worker")


def _get_app(app_path: str) -> typing.Callable:
    fp, app_name = app_path.split(":")
    fp = fp.replace("/", ".").replace("\\", ".")
    module = importlib.import_module(fp)
    app = getattr(module, app_name, None)
    if app is None:
        raise ImportError("No app named {} in file {}".format(app_name, fp))
    return app


class Worker:
    def __init__(
            self,
            app: str,
            port: int,
            shard_count: int,
            authorization: str,
            adapter: typing.Union[WSGIAdapter, ASGIAdapter, RawAdapter],
    ):
        self._app = _get_app(app)

        self._free_port = port

        worker_addr = "ws://127.0.0.1:{}/workers".format(self._free_port)
        logger.info("Binding to {}".format(worker_addr))
        self._shard_count = shard_count
        self.shard_manager = AutoShardedWorker(
            worker_addr,
            self._on_http_request,
            self._on_internal_message,
            authorization,
            shard_count=shard_count
        )

        self._adapter = adapter

    async def run(self) -> None:
        await self.shard_manager.run()

    async def _on_http_request(self, ws: ClientWebSocketResponse, msg: dict) -> None:
        await self._adapter(ws, self._app, msg)

    async def _on_internal_message(self, ws: ClientWebSocketResponse, msg: str) -> None:
        pass
