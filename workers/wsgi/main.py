import sys
import websocket
import logging
import json
import typing

from io import StringIO

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

        caller = StartResponse()
        wsgi_resp = self._wsgi_app(environ, caller)
        body = caller.join_body(wsgi_resp)
        ws.send(json.dumps({'id': resp['id'], 'content': body}))


class StartResponse:
    def __call__(self, *args, **kwargs):
        pass
        # print(args, kwargs)

    @staticmethod
    def join_body(resp: typing.Generator):
        body = ""
        for bod in resp:
            body += bod.decode('utf-8')
        return body


def _format_headers(headers: dict):
    return dict(map(lambda v: ("HTTP_%s" % v[0], v[1]), headers.items()))

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
