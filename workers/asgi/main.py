
from .. import connector


class AsgiAdapter(connector.Server):
    def __init__(self):
        super().__init__()


if __name__ == '__main__':
    wsgi = WsgiAdapter()