import argparse

from .workers.websocket import AutoShardedWorker, WebsocketShard, InternalResponses
from .workers.workers import Worker
from .main import run
from .adapters.asgi import ASGIAdapter
from .adapters.wsgi import WSGIAdapter
from .adapters.raw import RawAdapter

flags = argparse.ArgumentParser()
flags.add_argument("--child", type=bool, required=True)
flags.add_argument("--app", type=str, required=True)
flags.add_argument("--adapter", type=str, required=True)
flags.add_argument("--port", type=int, required=True)
flags.add_argument("--shards", type=int, required=True)
parsed = flags.parse_args()


def is_child() -> bool:
    return parsed.child
