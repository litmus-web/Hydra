[![CodeFactor](https://www.codefactor.io/repository/github/project-dream-weaver/sandman/badge)](https://www.codefactor.io/repository/github/project-dream-weaver/sandman)

# The universal production server for python
Sandman aims to provide a unviseral fit for WSGI/ASGI or any other interface for frameworks through a mutual connection while also boosting performance and adding the ability for stateful worker processes.<br>

Compatible with both Windows and POSIX operating systems providing top performance for both systems regardless of uvloop or other speedup limitations.<br>

## How does it work?
Sandman has a main server (acting in place of Nginx or Apache but can operate behind them as well) written in Go lang using Fast HTTP, any http requests then are sent to python workers as Websocket requests with the aim to remove any heavy lifting from Python to allow for faster applications and lighter tasks, typically running with a lower average latency and higher requests a second.

## What can it do?
- Serve static files with a production grade compiled server.
- Accelerate existing WSGI and ASGI systems.
- Provide a common interface for any framework, one server fits all.
- Add a global buffer for requests resulting in fewer socket errors.
- Faster request times with the power of Websockets.
- Minimal dependancies.
- Windows, Linux and MacOS compatible, anything that runs on windows runs on linux without the need for changing anything on the server and vice versa.


