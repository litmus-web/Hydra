import typing
import asyncio
import logging

from concurrent.futures import ThreadPoolExecutor
from datetime import datetime
from subprocess import Popen, PIPE, STDOUT

from ..workers import Worker


logger = logging.getLogger("Sandman-Worker")


class WindowsWorker(Worker):
    def __init__(
            self,
            app: str,
            host_addr: str = "127.0.0.1",
            port: int = 8000,
            shard_count: int = 1,
            sandman_path: str = "sandman"
    ):
        super().__init__(
            app,
            host_addr,
            port,
            shard_count,
            sandman_path,
            self.shard_manager_shutdown
        )

        # Process and thread management for win api
        self._active_process: typing.Optional[Popen[str]] = None
        self._shutdown_thread = False

    async def __aenter__(self) -> 'WindowsWorker':
        return self

    async def __aexit__(self, _exc_type, _exc_val, _exc_tb):
        self._shutdown_thread = True
        try:
            await asyncio.wait_for(self._has_shutdown(), timeout=10)
        except asyncio.TimeoutError:
            self.force_shutdown_unsafe()

    async def _has_shutdown(self) -> None:
        if self._active_process is None:
            return
        while self._active_process.poll() is None:
            await asyncio.sleep(0.5)

    def force_shutdown_unsafe(self) -> None:
        self._active_process.terminate()

    async def _spawn_rust(self) -> None:
        asyncio.create_task(self.manage_subprocess())

    async def shard_manager_shutdown(self):
        self._shutdown_thread = True
        try:
            await asyncio.wait_for(self._has_shutdown(), timeout=10)
        except asyncio.TimeoutError:
            self.force_shutdown_unsafe()

    async def manage_subprocess(self) -> None:
        self._active_process = Popen(
            "{sp} {worker_port} {bind_addr} {bind_port}".format(
                sp=self._exe_path,
                worker_port=self._free_port,
                bind_addr=self.host_addr,
                bind_port=self.port,
            ),
            shell=True,
            stdout=PIPE,
            stderr=STDOUT
        )

        self._clear_to_shard = True
        with ThreadPoolExecutor() as pool:
            while (not self._shutdown_thread) and (self._active_process.poll() is None):
                try:
                    out = await asyncio.wait_for(self.read_active_process(pool), timeout=5)
                except asyncio.TimeoutError:
                    continue

                if out.replace(b'\n', b'') != b'':
                    if b"is not recognized as an internal or external command" in out:
                        self._active_process.kill()
                        raise RuntimeError(out.decode())
                    print("[ %s ][ Sandman Message ] %s" % (
                        str(datetime.now())[:-2], out.decode().replace("\n", "")))
                await asyncio.sleep(0.2)
        self._clear_to_shard = False
        self._active_process.kill()

    async def read_active_process(self, pool: ThreadPoolExecutor):
        return await asyncio.get_event_loop().run_in_executor(
            pool, self._active_process.stdout.readline)
