import asyncio
import json
import sys

from aiohttp import ClientSession, WSMsgType, WSMessage


if len(sys.argv) > 1:
    _,  app, id_, auth = sys.argv
    print(app, id_, auth)
else:
    app = lambda: ""
    id_ = 1
    auth = "XVlBzgbaiCMR"

class Server:
    def __init__(self, worker_id: int, addr: str):
        self._host: str = addr
        self._ws = None
        self._id = worker_id
        self._loop = asyncio.get_event_loop()
        self._auth = auth

    def start(self) -> None:
        asyncio.get_event_loop().run_until_complete(self._start())

    async def _start(self) -> None:
        async with ClientSession() as session:
            self._ws = await session.ws_connect(self._host)
            while True:
                data = await self._ws.receive()
                if data.type == WSMsgType.CLOSED:
                    break
                elif data.type == WSMsgType.PING:
                    await self._ws.send_str(WSMessage(type=WSMsgType.PONG, data="Hello world!"))
                elif data.type == WSMsgType.TEXT:
                    self._loop.create_task(self._handle_connections(data.data))
                else:
                    raise TypeError("Unknown WS data type has been sent.")
            raise ConnectionError("Worker {} has lost connection to sandman.".format(self._id))

    async def _handle_connections(self, data: str):
        if data == "WORKER.IDENTIFY":
            return await self._ws.send_json({"authorization": self._auth, "id_": str(self._id)})
        payload = json.loads(data)
        await self.on_message(self._ws, payload)

    @staticmethod
    async def on_message(ws, msg: dict) -> None:
        resp = {
            "body": "hello world",
            "headers": {},
            "status": 200
        }
        await ws.send_json(resp)


if __name__ == '__main__':
    s = Server(worker_id=int(id_), addr="ws://127.0.0.1/workers")

    try:
        s.start()
    except KeyboardInterrupt:
        asyncio.get_event_loop().stop()
        asyncio.get_event_loop().close()