import sys

platform = sys.platform

if platform.startswith("win32") or platform.startswith("cygwin"):
    from .win import WindowsWorker as Worker
else:
    from .unix import UnixWorker as Worker
