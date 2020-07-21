import asyncio
import time
import socket
import json

with open('index.html') as file:
    html = file.read()


class MySocketClient:
    def __init__(self, host="127.0.0.1", port=9090, **options):
        self._host = host
        self._port = port
        self._loop = options.get('loop', asyncio.get_event_loop())

        self._client = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._client.connect((self._host, self._port))
        self._client.setblocking(False)
        self._callback = options.get('callback')

        print("======= Connected to socket on 127.0.0.1:9090 ======")
        print("🚀 Hello from Rust!")

    @staticmethod
    def _handle_content(content):
        if content.endswith("--eof--"):
            return True, content.strip("--eof--")
        elif content != "--eof--":
            return False, content.strip("--eof--")
        return True, ""

    async def _get_resp(self):
        full_content = ""
        complete = False
        while not complete:
            try:
                content = (await self._loop.sock_recv(self._client, 1024)).decode('utf8')
            except (ConnectionResetError, ConnectionAbortedError):
                return None
            else:
                complete, content = self._handle_content(content)
                full_content += content
        return full_content

    async def start(self):
        running = True
        while running:
            content = await self._get_resp()
            if content is not None:
                self._loop.create_task(self._trigger_event(content))
            else:
                running = False

    async def _trigger_event(self, msg):
        resp = await self._callback(msg)
        await self._send(resp)

    async def _send(self, message):
        await self._loop.sock_sendall(self._client, message.encode('utf-8') + b"--eom--")

    async def close(self):
        await self._loop.sock_sendall(self._client, b"--quit--")
        await asyncio.sleep(2)
        self._client.close()


async def on_message(request):
    return html


if __name__ == '__main__':
    my_server = MySocketClient(callback=on_message)
    loop = asyncio.get_event_loop()
    try:
        loop.run_until_complete(my_server.start())
    except (SystemExit, KeyboardInterrupt):
        print("Shutting down gracefully.")
        loop.run_until_complete(my_server.close())
    loop.close()
