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

## How well does is perform?
Although i have only done testing benchmarks the results are promising! On a windows system Sandman averages a **125%** requests/sec increase when benching against [Uvicorn](https://www.uvicorn.org/), Although this is windows and nothing is ever as good as it could be :P

On linux the last tests we did we averaged about a **30%** request/sec increase with [Uvicorn](https://www.uvicorn.org/) running with Uvloop and Sandman running on pure python with zero speed ups (Sandman is based off of AyncIO and would also benifit heavily from AioHTTP's speedups option and uvloop installs).


**These are the following results when tested with Wrk on windows, take this with a pinch of salt because there are some massive bottleknecks here.**
```docker
============= Sandman =============
 Running 2m test @ http://127.0.0.1:5000/
   2 threads and 256 connections
   Thread Stats   Avg      Stdev     Max   +/- Stdev
     Latency    31.78ms   40.64ms   1.16s    95.92%
     Req/Sec     4.88k     0.86k    6.33k    87.94%
   1163658 requests in 2.00m, 163.13MB read
 Requests/sec:   9691.59
 Transfer/sec:      1.36MB
 
============= Uvicorn =============
 Running 2m test @ http://127.0.0.1:5050/
   2 threads and 256 connections
   Thread Stats   Avg      Stdev     Max   +/- Stdev
     Latency    64.69ms   37.38ms 880.21ms   96.23%
     Req/Sec     2.11k   386.98     2.52k    91.68%
   502029 requests in 2.00m, 71.84MB read
 Requests/sec:   4181.81
 Transfer/sec:    612.77KB
```
