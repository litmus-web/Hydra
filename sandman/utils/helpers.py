import socket
import argparse

from contextlib import closing


parser = argparse.ArgumentParser()
parser.add_argument("--child", required=False, default=False, type=bool)
sys_args = parser.parse_args()


def find_free_port() -> int:
    with closing(socket.socket(socket.AF_INET, socket.SOCK_STREAM)) as sock:
        sock.bind(('', 0))
        sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        return sock.getsockname()[1]


def is_child() -> bool:
    return sys_args.child


