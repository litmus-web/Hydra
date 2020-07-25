import websocket
import pyuv
import typing
import json


addr = "ws://127.0.0.1:8080/workers"


class Server:
    def __init__(self, worker_id: int, callback: typing.Callable):
        self._ws = websocket.WebSocketApp(
            addr,
            on_message=self.on_message,
            on_error=self.on_error,
            on_close=self.on_close
        )
        self._worker_id = worker_id
        self._callback = callback
        self._is_ready = False

        self._loop: pyuv.Loop = pyuv.Loop.default_loop()
        self._loop.run(pyuv.UV_RUN_NOWAIT)

    def start(self) -> None:
        self._ws.run_forever()

    def on_message(self, ws: websocket.WebSocket, message: str) -> None:
        if not self._is_ready and message == "HELLO.WORLD":
            ws.send(json.dumps({'worker_id': str(self._worker_id)}))
            print("Sent opening payload to Sandman")
        else:
            task = Task(self._callback, ws, message)
            self._loop.queue_work(task)

    def on_error(self, ws: websocket.WebSocket, error) -> None:
        print(error)

    def on_close(self, ws: websocket.WebSocket) -> None:
        print("Worker lost connection to Sandman.")


class Task:
    def __init__(self, handle: typing.Callable, *args, **kwargs):
        self._handle = handle
        self._args = args
        self._kwargs = kwargs

    def __call__(self, *args, **kwargs):
        return self._handle(*self._args, **self._kwargs)


if __name__ == '__main__':
    def handle_incoming(ws: websocket.WebSocket, msg: str):
        resp = json.loads(msg)
        ws.send(json.dumps({'id': resp['id'], 'content': "content['body']"}))
    server = Server(1, handle_incoming)
    server.start()
