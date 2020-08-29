import aiohttp
import asyncio
import typing
import logging
import os

from typing import Any
from dataclasses import dataclass
from aiohttp import WSMessage, WSMsgType, ClientConnectionError

from ..codes import OpCodes
from ..helpers import dumps_data, load_data, JSONDecodeError


__all__ = ["WebsocketShard", "AutoShardedWorker", "InternalResponses"]

logger = logging.getLogger("Sandman-Shard")
PID = os.getpid()


def log_info(msg, *args):
    logger.info("[ Worker %s ][ Worker Shard ] %s", "{}".format(PID).ljust(5), msg, *args)


class ConnectionFailed:
    """Signifies the worker failing to connect"""


class ClosedNaturally:
    """Signifies the worker being disconnected by Sandman"""


class ClosedAbnormally:
    """Signifies the worker loosing connection during a request due to a websocket error."""


@dataclass(frozen=True)
class InternalResponses:
    CONNECTION_FAILED = ConnectionFailed()
    CLOSED_NATURALLY = ClosedNaturally()
    CLOSED_ABNORMALLY = ClosedAbnormally()


class WebsocketShard:
    """This class represents a single websocket shard that connects to Sandman
    that then handles incoming HTTP requests.

    Parameters
    -----------
    shard_id: :class:`int`
        The specific shard id of the object used in identification.
    binding_addr: :class:`str`
        The websocket address for AioHTTP to bind to.
    request_callback: Union[Coroutine, Callable]
        A Coroutine function or callable to handle incoming HTTP requests.
    msg_callback: Union[asyncio.coroutine, Callable]
        A Coroutine function or callable to handle any messages
        from the WS that are not HTTP requests.
    """

    def __init__(
            self,
            shard_id: int,
            binding_addr: str,
            request_callback: typing.Union[typing.Coroutine[Any, Any, None], typing.Callable],
            msg_callback: typing.Union[typing.Coroutine[Any, Any, None], typing.Callable],
    ):
        self.shard_id = shard_id
        self.binding_addr = binding_addr
        self.req_callback = request_callback
        self.msg_callback = msg_callback
        self.session: typing.Optional[aiohttp.ClientSession] = None
        self.loop = asyncio.get_event_loop()

    async def connect(self) -> typing.Union[ConnectionFailed, ClosedNaturally, ClosedAbnormally]:
        """Connects to the worker socket on Sandman and begins receiving requests"""
        self.session = aiohttp.ClientSession()
        log_info("Shard initiated client session")
        try:
            async with self.session.ws_connect(self.binding_addr) as ws:
                await self.on_connect(ws)

                while not ws.closed:
                    msg: WSMessage = await ws.receive()
                    if msg.type == WSMsgType.TEXT:
                        self.loop.create_task(self.on_message(ws, msg))
                    elif msg.type == WSMsgType.CLOSE:
                        await self.on_close(ws, msg)
                    elif msg.type == WSMsgType.CLOSED:
                        await self.on_close(ws, msg)
                    elif msg.type == WSMsgType.ERROR:
                        await self.on_error(ws, msg)

            await self.session.close()
            if msg.type == WSMsgType.CLOSED:
                return InternalResponses.CLOSED_NATURALLY
            return InternalResponses.CLOSED_ABNORMALLY

        except (aiohttp.ClientConnectionError, aiohttp.WSServerHandshakeError):
            await self.session.close()
            return InternalResponses.CONNECTION_FAILED

    async def on_connect(self, _) -> None:
        """Coroutine called when a connection has been established between the
        worker shard and Sandman.
        """
        log_info("Shard has connected to Sandman")

    async def on_error(self, ws, _: WSMessage) -> None:
        """Coroutine called when a connection has been interrupted by a error,
        by default this closes the websocket and the AioHTTP session.
        """
        await ws.close()
        await self.session.close()

    async def on_close(self, ws, _) -> None:
        """Coroutine called when a connection has been closed by Sandman directly,
        by default this closes the websocket and the AioHTTP session.

        Parameters
        -----------
        ws: :class:`aiohttp.ClientWebSocketResponse`
            A AioHTTP websocket session provided by the loop.
        """
        await ws.close()
        await self.session.close()
        log_info("websocket has closed, shutting down shard.")

    async def on_message(self, ws, msg: WSMessage) -> None:
        """Coroutine called when a connection has been closed by Sandman directly,
        by default this closes the websocket and the AioHTTP session.]

        Parameters
        -----------
        ws: :class:`aiohttp.ClientWebSocketResponse`
            A AioHTTP websocket session provided by the loop.
        msg: :class:`aiohttp.WSMessage`
            A websocket message object with the type TEXT.
        """
        try:
            data = load_data(msg.data)
        except JSONDecodeError:
            if asyncio.iscoroutinefunction(self.msg_callback):
                return await self.msg_callback(ws, msg.data)
            return self.msg_callback(ws, msg.data)

        try:
            if data["op"] == 0:
                ident = {
                    "op": OpCodes.IDENTIFY,
                    "shard_id": self.shard_id
                }
                await ws.send_json(ident)
            elif data["op"] == 1:
                await self.req_callback(ws, data)

        except Exception as err:
            if data.get('op', -1) == 1:
                data = {
                    "op": OpCodes.HTTP_REQUEST,
                    "request_id": data["request_id"],
                    "status": 503,
                    "headers": [],
                    "body": "A internal server error has occurred."
                }
                await ws.send_bytes(dumps_data(data))
                raise err
            else:
                print(data)


class AutoShardedWorker:
    """This class represents a sharding manager for a worker process, it spawns and manages
    shards connecting to Sandman to balance load between connections.

    Parameters
    -----------
    binding_addr: :class:`str`
        The websocket address for AioHTTP to bind to.
    request_callback: Union[Coroutine, Callable]
        A Coroutine function or callable to handle incoming HTTP requests.
    msg_callback: Union[asyncio.coroutine, Callable]
        A Coroutine function or callable to handle any messages
        from the WS that are not HTTP requests.
    shard_count: Optional[:class:`int`]
        The amount of shards / sessions the worker process should open with Sandman.
        Defaults to 1 which is generally fine but larger messages may require more.
    """

    def __init__(
            self,
            binding_addr: str,
            request_callback: typing.Union[typing.Coroutine[Any, Any, None], typing.Callable],
            msg_callback: typing.Union[typing.Coroutine[Any, Any, None], typing.Callable],
            shard_count: int = 1,
            shard_restart_limit: typing.Optional[int] = None,
    ):
        self.shard_count = shard_count
        self.binding_addr = binding_addr
        self.req_callback = request_callback
        self.msg_callback = msg_callback

        if shard_restart_limit is None:
            self.shard_restart_limit = shard_count * 2
        else:
            self.shard_restart_limit = shard_restart_limit

        self._shards = {}
        self._shard_restarts = 0
        self._loop = asyncio.get_event_loop()

    async def run(self) -> None:
        """Creates n amount of shards and manages them as they connect,
        the shards start 0 -> n where n is shard_count - 1, after all
        shards are spawned the server the watcher is ran to keep shards
        alive if they die.
        """
        for i in range(self.shard_count):
            self._create_shard(shard_id=i)
        await self._check_shards()

    def _create_shard(self, shard_id) -> asyncio.Task:
        shard = WebsocketShard(
            shard_id=shard_id,
            binding_addr=self.binding_addr,
            request_callback=self.req_callback,
            msg_callback=self.msg_callback,
        )
        task = self._loop.create_task(shard.connect())
        self._shards[shard_id] = task
        return task

    async def _check_shards(self) -> None:
        continue_checking = True
        while continue_checking:
            for shard_id, task in self._shards.items():
                task: asyncio.Task = task
                if task.done():
                    res = task.result()
                    if res == InternalResponses.CLOSED_NATURALLY:
                        await self._shutdown_all()
                        break
                    elif res == InternalResponses.CONNECTION_FAILED:
                        await self._shutdown_all()
                        raise ClientConnectionError(
                            "Worker failed to Connect to WS on addr: {}".format(self.binding_addr))
                    else:
                        self._create_shard(shard_id=shard_id)
                        logger.warning(
                            "Shard manager has restarted shard %s due to a "
                            "exception raised by the WS", shard_id)
                        self._shard_restarts += 1
                        await asyncio.sleep(1)
            else:
                await asyncio.sleep(0.5)
                continue
            continue_checking = False

    async def _shutdown_all(self) -> None:
        for shard_id, task in self._shards.items():
            task: asyncio.Task = task
            if not task.done() and not task.cancelled():
                task.cancel()


# Testing
async def main():
    test = WebsocketShard(
        shard_id=0,
        binding_addr="ws://127.0.0.1:12232/workers",
        request_callback=lambda x: print(x),
        msg_callback=lambda x: print(x)
    )
    await test.connect()


if __name__ == '__main__':
    logging.basicConfig(level=logging.INFO)
    asyncio.run(main())
