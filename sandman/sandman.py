import argparse
import typing
import asyncio
import importlib

parser = argparse.ArgumentParser()

parser.add_argument("-host", "--hostaddr", dest="host", help="Binding address", default="127.0.0.1")
parser.add_argument("-p", "--port", dest="port", help="Binding port", default="8080")
parser.add_argument("-w", "--workers", dest="worker_count", help="Amount of workers", default=1, type=int)

args = parser.parse_args()


class Sandman:
    def __init__(
            self,
            file_path: typing.Optional[str] = None,
            app_path: typing.Optional[str] = None,
            host: typing.Optional[str] = None,
            port: typing.Optional[int] = None,
            worker_count: typing.Optional[int] = None
    ):
        """Starts the server programmatically.

        Args:
            file_path (str): A path to a python file containing a ASGI, WSGI or Raw connector.
            app_path (str): The name of a ASGI, WSGI or Raw connector.
            host (str): A string representing a binding address e.g 127.0.0.1.
            port (int): A integer representing the binding port, Ports < 1024 require admin perms.
            worker_count (int): The number of workers sandman should start up.

        returns:
            None
        """
        self._fp = file_path,
        self._app_path = app_path
        self._host = host,
        self._port = port,
        self._worker_count = worker_count

    def run(
            self,
            target: str,
            host: typing.Optional[str] = "127.0.0.1",
            port: typing.Optional[int] = 8080,
            worker_count: typing.Optional[int] = None
    ) -> None:
        """Starts the server programmatically.

        Parameters
        -----------
        target: :class:`str`
            A string labeling the filepath and target callable separated by ``:``.
        host: Optional[:class:`str`]
            A string representing a binding address e.g ``127.0.0.1``.
        port: Optional[:class:`int`]
            A integer representing the binding port, Ports < 1024 require admin perms.
        worker_count: Optional[:class:`int`]
            The number of workers sandman should start up.
        """

        target_secs = target.split(":")

        self._fp: typing.Optional[str] = target_secs[0]
        self._app_path: typing.Optional[str] = target_secs[1]
        self._host: typing.Optional[str] = host
        self._port: typing.Optional[int] = port
        self._worker_count: typing.Optional[int] = worker_count

        self.start()

    def start(self) -> None:
        """Starts the event loop and the server, invoked directly if the system is ran via cli.

        Returns:
          None
        """
        asyncio.get_event_loop().run_until_complete(self.__run_internal())

    async def __run_internal(self):
        pass


if __name__ == "__main__":
    Sandman(args.host, args.port, args.worker_count)
