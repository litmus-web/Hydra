import socket
import time

UDP_IP = "127.0.0.1"
UDP_PORT = 5005

MESSAGE = b"Hello, World!" * 500_000

print("UDP target IP: %s" % UDP_IP)
print("UDP target port: %s" % UDP_PORT)
print("message: %s" % MESSAGE)

sock = socket.socket(
    socket.AF_INET,  # Internet
    socket.SOCK_DGRAM  # UDP
)

start = time.perf_counter()
sock.sendto(MESSAGE, (UDP_IP, UDP_PORT))
stop = time.perf_counter() - start

print("took {}ms".format(stop * 1000))