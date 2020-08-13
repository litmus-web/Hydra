import asyncio
import sys

from datetime import datetime

import typing

try:
    import uvloop
    uvloop.install()
except ImportError:
    pass

from . import is_child, Worker
from .workers.manage_workers import ProcessManager
from .adapters.asgi import ASGIAdapter
from .adapters.wsgi import WSGIAdapter
from .adapters.raw import RawAdapter

platform = sys.platform


def run(
        app: str,
        adapter: typing.Union[WSGIAdapter, ASGIAdapter, RawAdapter],
        host_addr: str = "127.0.0.1",
        port: int = 8000,
        workers: int = 1,
        shards_per_proc: int = 1,
        sandman_path: str = "sandman"
):
    print("[ %s ][ Sandman ] %s" % (
        str(datetime.now())[:-2], "Starting server on http://{}:{}".format(host_addr, port)))

    #adapter.server_info.port = port
    # Windows does not support SO_REUSEADDR on Tokio.rs's side.
    if platform.startswith("win32") or platform.startswith("cygwin"):
        if workers != 1:
            print("[ %s ][ Sandman ] %s" % (
                str(datetime.now())[:-2],
                "Multiple workers are not supported by Tokio.rs on Windows. Defaulting to 1."))
        worker = Worker(
            app=app,
            host_addr=host_addr,
            port=port,
            shard_count=shards_per_proc,
            sandman_path=sandman_path,
            adapter=adapter
        )
        return asyncio.get_event_loop().run_until_complete(worker.run())

    if is_child():
        worker = Worker(
            app=app,
            host_addr=host_addr,
            port=port,
            shard_count=shards_per_proc,
            sandman_path=sandman_path,
            adapter=adapter
        )
        return asyncio.get_event_loop().run_until_complete(worker.run())

    manager = ProcessManager(num_workers=workers)
    return asyncio.get_event_loop().run_until_complete(manager.start())
