import typing

from asyncio import subprocess
from asyncio.subprocess import Process

from ..workers import Worker

class UnixWorker(Worker):
    def __init__(
            self,
            app: str,
            host_addr: str = "127.0.0.1",
            port: int = 8000,
            shard_count: int = 1,
            sandman_path: str = "sandman"
    ):
        super().__init__(app, host_addr, port, shard_count, sandman_path)
        self.active_process: typing.Optional[Process] = None

    async def _spawn_children(self):
        pass

    async def _spawn_rust(self):
        pass