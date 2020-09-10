import typing as t

try:
    import orjson as json
    from orjson import JSONDecodeError

    def dumps_data(data):
        return json.dumps(data)     # This does actually return bytestr its just badly labeled.

except ImportError:
    import json
    from json import JSONDecodeError

    def dumps_data(data):
        return json.dumps(data).encode()


def load_data(data: str):
    return json.loads(data)
