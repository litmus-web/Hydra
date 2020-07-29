import asyncio
from aiohttp import ClientSession


class Server:
    def __init__(self, worker_id: int, addr: str):
        self._host: str = addr
        self._ws = None
        self._id = worker_id
        self._loop = asyncio.get_event_loop()

    def start(self) -> None:
        asyncio.get_event_loop().run_until_complete(self._start())

    async def _start(self) -> None:
        async with ClientSession() as session:
            self._ws = await session.ws_connect(self._host)
            print("Worker {} has connected to Sandman!".format(self._id))
            while True:
                resp = (await self._ws.receive_str())
                self._loop.create_task(self.on_message(self._ws, resp))

    async def on_message(self, ws, msg) -> None:
        resp = {
            "body": "hello world",
            "headers": {},
            "status": 200
        }
        await ws.send_json(resp)


if __name__ == '__main__':
    s = Server(worker_id=1, addr="ws://127.0.0.1/workers")
    s.start()