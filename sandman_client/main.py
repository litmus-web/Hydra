import asyncio

try:
    import uvloop
    uvloop.install()
except ImportError:
    pass

from . import Worker, parsed
from .adapters.asgi import ASGIAdapter
from .adapters.wsgi import WSGIAdapter
from .adapters.raw import RawAdapter


adapters = {
    'asgi': ASGIAdapter,
    'wsgi': WSGIAdapter,
    'raw': RawAdapter,
}


def run():
    adapter_str = parsed.adapter
    adapter = adapters.get(adapter_str, RawAdapter)

    worker = Worker(
        app=parsed.app,
        port=parsed.port,
        shard_count=parsed.shards,
        adapter=adapter()
    )
    return asyncio.get_event_loop().run_until_complete(worker.run())
