import asyncio
import typing

from aiohttp import ClientSession, ClientConnectionError


class Server:
    def __init__(self, worker_id: int, addr: str, app: typing.Callable):
        self._host: str = addr
        self._ws = None
        self._id = worker_id
        self._app = app

    def start(self) -> None:
        asyncio.get_event_loop().run_until_complete(self._start())

    async def _start(self) -> None:
        async with ClientSession() as session:
            attempts, connected = 0, False
            while not connected:
                try:
                    self._ws = await session.ws_connect(self._host)
                    connected = True
                except ClientConnectionError:
                    for i in range(3):
                        print("Failed to connect to WS, Retrying in {}".format(3 - i))
                        await asyncio.sleep(1)
                    continue
                finally:
                    attempts += 1
                    if attempts >= 3:
                        break

            if not connected:
                raise ConnectionRefusedError("Cannot connect to Sandman on {}".format(self._host))

            resp: str = (await self._ws.receive_str())
            if resp != "HELLO.WORLD":
                raise ValueError("Worker has connected and received a invalid startup message.")
            await self._ws.send_json({'worker_id': str(self._id)})
            print("Worker {} has connected to Sandman!".format(self._id))

            while True:
                resp: dict = (await self._ws.receive_json())
                await self.on_message(self._ws, resp)

    async def on_message(self, ws, msg: dict) -> None:
        resp_id = msg['id']
        ctx: dict = msg['context']

        scope = _format_scope(ctx)
        recv = Receive("")
        send = Send(ws, resp_id)

        asyncio.create_task(self._app(scope, recv, send))

def _format_scope(ctx: dict) -> dict:
    addr, port = ctx['headers']['host'].split(":")
    return {
        'type': 'http',
        'asgi': {'spec_version': '2.1', 'version': '3.0'},

        'scheme': 'http',
        'http_version': '1.1',

        'server': ('127.0.0.1', 8080),
        'client': (addr, int(port)),
        'headers': list(
                map(
                    lambda x: (
                        x[0].encode('latin1'),
                        x[1].encode('latin1')
                    ),
                    ctx['headers'].items()
                )
            ),

        'method': ctx['method'],
        'query_string': b'',

        'path': ctx['path'],
        'root_path': '',
        'raw_path': ctx['path'].encode('utf-8'),
    }


class Receive:
    def __init__(self, content):
        self._content = content
        self._sections = [
            {
                'type': 'http.request',
                'body': self._content.encode('utf-8'),
                'more_body': False
            },
            {
                'type': 'http.disconnect'
            },
        ].__iter__()

    def __call__(self, *args, **kwargs) -> asyncio.coroutine:
        return self.call_me()

    async def call_me(self):
        return next(self._sections)


class Send:
    def __init__(self, ws, id_):
        self._id = id_
        self._ws = ws
        self._body = ""
        self._status_code = 200
        self._headers = []

    def __call__(self, *args, **kwargs) -> asyncio.coroutine:
        ctx = args[0]
        if ctx['type'] == 'http.response.start':
            self._status_code = ctx['status']
            self._headers = list(
                map(
                    lambda x: (
                        x[0].decode('latin1'),
                        x[1].decode('latin1')
                    ),
                    ctx['headers']
                )
            )
            return self._dud()
        elif ctx['type'] == 'http.response.body':
            self._body += ctx['body'].decode('utf-8')
            if not ctx.get('more_body', False):
                return self._send_to_sock()
        return self._dud()

    async def _send_to_sock(self):
        await self._ws.send_json(
            {
                'id': self._id,
                'body': self._body,
                'status': self._status_code,
                'headers': self._headers,
            }
        )

    async def _dud(self):
        return
