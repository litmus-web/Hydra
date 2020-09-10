import asyncio
import argparse
import typing as t

try:
    import uvloop
    uvloop.install()
except ImportError:
    uvloop = None

from . import Worker
from .adapters.asgi import ASGIAdapter
from .adapters.wsgi import WSGIAdapter
from .adapters.raw import RawAdapter


flags = argparse.ArgumentParser()
flags.add_argument("--child", type=bool, required=True)
flags.add_argument("--app", type=str, required=True)
flags.add_argument("--adapter", type=str, required=True)
flags.add_argument("--port", type=int, required=True)
flags.add_argument("--shards", type=int, required=True)


adapters = {
    'asgi': ASGIAdapter,
    'wsgi': WSGIAdapter,
    'raw': RawAdapter,
}


def run() -> None:
    parsed = flags.parse_args()
    adapter_str = parsed.adapter
    adapter = adapters.get(adapter_str, RawAdapter)

    worker = Worker(
        app=parsed.app,
        port=parsed.port,
        shard_count=parsed.shards,
        adapter=adapter()
    )
    return asyncio.get_event_loop().run_until_complete(worker.run())
