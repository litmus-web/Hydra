from dataclasses import dataclass


@dataclass(frozen=True)
class OpCodes:
    IDENTIFY = 0
    HTTP_REQUEST = 1
    MESSAGE = 2