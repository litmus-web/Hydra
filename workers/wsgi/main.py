import sys
import websocket
import logging
import json
import typing

from pprint import pprint
from io import StringIO

if __name__ == '__main__':
    import connector
else:
    from . import connector


logger = logging.getLogger("Sandman")


class WsgiAdapter(connector.Server):
    def __init__(self, wsgi_app):
        sys_args = sys.argv
        if len(sys_args) < 2:
            self._worker_id = 1
            logger.warning(
                "WARNING: Worker running in MANUAL mode, worker id has defaulted to 1.\n"
                "This can effect the main Sandman server.\n"
                "Please do NOT use this method if you are running this outside of developing Sandman.\n"
            )
        else:
            self._worker_id = int(sys_args[1])
        super().__init__(worker_id=self._worker_id, callback=self.on_http_req)

        self._wsgi_app = wsgi_app

    def on_http_req(self, ws: websocket.WebSocket, msg: str):
        resp = json.loads(msg)

        environ = _to_environ(resp['context'])

        caller = StartResponse(ws, resp['id'])
        wsgi_resp = self._wsgi_app(environ, caller)
        caller.join_body(wsgi_resp).send()


class StartResponse:
    def __init__(self, ws: websocket.WebSocket, id_: int):
        self._ws = ws
        self._id = id_
        self._body = ""
        self._status_code = 200
        self._headers = []

    def __call__(self, *args, **kwargs):
        status: str = args[0]
        self._status_code = int(status.split(" ")[0])
        self._headers = args[1]

    def join_body(self, resp: typing.Generator):
        for bod in resp:
            self._body += bod.decode('utf-8')
        return self

    def send(self):
        self._ws.send(
            json.dumps(
                {
                    'id': self._id,
                    'body': self._body,
                    'status': self._status_code,
                    'headers': self._headers,
                }
            )
        )


def _format_headers(headers: dict):
    return dict(map(lambda v: ("HTTP_%s" % v[0].upper(), v[1]), headers.items()))

def _to_environ(context: dict) -> dict:
    return {
        'SERVER_PORT': context['port'],
        'SERVER_NAME': context['server'],
        'REQUEST_METHOD': context['method'],
        'PATH_INFO': context['path'],
        'SERVER_PROTOCOL': "HTTP/1.1",
        'wsgi.input': StringIO(""),
        'wsgi.url_scheme': "",
        **_format_headers(context['headers'])
    }


if __name__ == '__main__':
    from flask import Flask
    app = Flask(__name__)

    @app.route("/")
    def hello_world():
        return "hello world"

    wsgi = WsgiAdapter(app)
    wsgi.start()