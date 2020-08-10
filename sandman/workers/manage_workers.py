import os
import typing
import asyncio
import sys

from asyncio.subprocess import Process
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime
from subprocess import Popen, PIPE, STDOUT


platform = sys.platform


def is_unix() -> bool:
    return not (platform.startswith("win32") or platform.startswith("cygwin"))


class WinProcess:
    def __init__(self, target: str):
        self._target = target
        self._pid: typing.Optional[int] = None
        self._active_process: typing.Optional[Popen] = None

    async def start(self) -> int:
        self._active_process = Popen(
            'python {fp} --child True'.format(fp=self._target),
            shell=True,
            stdout=PIPE,
            stderr=PIPE
        )
        self._pid = self._active_process.pid
        return self._active_process.pid

    async def stop(self):
        if self._active_process.poll() is None:
            self._active_process.kill()
        while self._active_process.poll() is None:
            await asyncio.sleep(0.5)

    async def logs(self):
        with ThreadPoolExecutor() as pool:
            while self._active_process.poll() is None:
                try:
                    msg = await asyncio.wait_for(self._get_logs(pool), timeout=5)
                    print(msg)
                except asyncio.TimeoutError:
                    break
                await asyncio.sleep(0.1)
            if self._active_process.poll() is not None:
                print(self._active_process.poll())
                print(self._active_process.communicate())

    async def _get_logs(self, pool):
        return await asyncio.get_event_loop()\
            .run_in_executor(pool, self._active_process.stdout.readline)

    @property
    def finished(self):
        return self._active_process.poll() is None

    @property
    def pid(self):
        return self._pid

    def __repr__(self):
        return "WinProcess(pid={})".format(self._pid)

    def __eq__(self, other):
        if not isinstance(other, WinProcess):
            raise ValueError("Only WinProcess objects are comparable.")
        return other._pid == other._pid


class UnixProcess:
    def __init__(self, target: str):
        self._target = target
        self._pid: typing.Optional[int] = None
        self._active_process: typing.Optional[Process] = None

    async def start(self) -> int:
        self._active_process = await asyncio.subprocess.create_subprocess_shell(
            'python {fp} --child True'.format(fp=self._target),
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.STDOUT,
        )
        self._pid = self._active_process.pid
        return self._active_process.pid

    async def stop(self):
        if self._active_process.returncode is None:
            self._active_process.kill()
        while self._active_process.returncode is None:
            await asyncio.sleep(0.5)

    async def logs(self) -> None:
        while self._active_process.returncode is None:
            try:
                msg = await asyncio.wait_for(
                    self._active_process.stdout.readline(), timeout=5)
                print(msg)
            except asyncio.TimeoutError:
                break

    @property
    def finished(self):
        return self._active_process.returncode is None

    @property
    def pid(self) -> int:
        return self._pid

    def __repr__(self) -> str:
        return "UnixProcess(pid={})".format(self._pid)

    def __eq__(self, other) -> bool:
        if not isinstance(other, UnixProcess):
            raise ValueError("Only UnixProcess objects are comparable.")
        return other._pid == other._pid


class ProcessManager:
    def __init__(self, num_workers: int):
        self._path: str = __file__
        self._num_workers = num_workers
        self._active_workers: typing.List[typing.Union[UnixProcess, WinProcess]] = []
        self._proc_class = UnixProcess if is_unix() else WinProcess

    async def start(self) -> None:
        await self._spawn_workers()
        await asyncio.sleep(1)
        await self._manage_workers()

    async def _spawn_workers(self) -> None:
        for _ in range(self._num_workers):
            proc = self._proc_class(self._path)
            await proc.start()
            self._active_workers.append(proc)

    async def _manage_workers(self) -> None:
        for proc in self._active_workers[:-1]:
            asyncio.create_task(proc.logs())
        await self._active_workers[len(self._active_workers) - 1].logs()
