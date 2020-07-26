import sys
import logging
import typing

from . import connector

logger = logging.getLogger("Sandman")


class ASGIAdapter(connector.Server):
    def __init__(self, app: typing.Callable):
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
        super().__init__(self._worker_id, "ws://127.0.0.1:8080/workers", app)
