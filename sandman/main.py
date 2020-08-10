import asyncio

from . import is_child, Worker
from .workers.manage_workers import ProcessManager


def run(
        app: str,
        host_addr: str = "127.0.0.1",
        port: int = 8000,
        workers: int = 1,
        shards_per_proc: int = 1,
        sandman_path: str = "sandman"
):

    if is_child():
        worker = Worker(
            app=app,
            host_addr=host_addr,
            port=port,
            shard_count=shards_per_proc,
            sandman_path=sandman_path
        )
        return asyncio.run(worker.run())

    manager = ProcessManager(num_workers=workers)
    return asyncio.get_event_loop().run_until_complete(manager.start())

