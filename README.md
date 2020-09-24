[![CodeFactor](https://www.codefactor.io/repository/github/project-dream-weaver/hydra/badge)](https://www.codefactor.io/repository/github/project-dream-weaver/hydra)

# The universal production server for python
Hydra aims to provide a unviseral fit for WSGI/ASGI or any other interface for frameworks through a mutual connection while also boosting performance and adding the ability for stateful worker processes.<br>

Compatible with both Windows and POSIX operating systems providing top performance for both systems regardless of uvloop or other speedup limitations.<br>

## How does it work?
Hydra has a main server (acting in place of Nginx or Apache but can operate behind them as well) written in Rust lang using Hyper, any http requests then are sent to python workers as Websocket requests with the aim to remove any heavy lifting from Python to allow for faster applications and lighter tasks, typically running with a lower average latency and higher requests a second.

## What can it do?
- Serve static files with a production grade compiled server.
- Accelerate existing WSGI and ASGI systems.
- Provide a common interface for any framework, one server fits all.
- Add a global buffer for requests resulting in fewer socket errors.
- Faster request times with the power of Websockets.
- Minimal dependancies.
- Windows, Linux and MacOS compatible, anything that runs on windows runs on linux without the need for changing anything on the server and vice versa.

## How well does it perform?
Although i have only done testing benchmarks the results are promising! On a windows system Hydra averages a **125%** requests/sec increase when benching against [Uvicorn](https://www.uvicorn.org/), Although this is windows and nothing is ever as good as it could be :P

On linux the last tests we did we averaged about a **35%** request/sec increase with [Uvicorn](https://www.uvicorn.org/) running with Uvloop and Hydra running on pure python with zero speed ups (Hydra is based off of AyncIO and would also benifit heavily from AioHTTP's speedups option and uvloop installs).


**These are the following results when tested with Wrk on linux**
```docker
wrk_test   | Starting Benchmark...
wrk_test   | ============= Hydra =============
wrk_test   | Running 25s test @ http://server:8080/
wrk_test   |   2 threads and 256 connections
wrk_test   |   Thread Stats   Avg      Stdev     Max   +/- Stdev
wrk_test   |     Latency     9.44ms    2.95ms 215.79ms   88.88%
wrk_test   |     Req/Sec    13.67k   841.55    15.01k    80.00%
wrk_test   |   680155 requests in 25.03s, 103.78MB read
wrk_test   | Requests/sec:  27169.62
wrk_test   | Transfer/sec:      4.15MB
wrk_test   | ============= Uvicorn =============
wrk_test   | Running 25s test @ http://uvicorn:5000/
wrk_test   |   2 threads and 256 connections
wrk_test   |   Thread Stats   Avg      Stdev     Max   +/- Stdev
wrk_test   |     Latency    13.48ms  764.26us  35.89ms   93.62%
wrk_test   |     Req/Sec     9.52k   529.57    10.34k    58.40%
wrk_test   |   474090 requests in 25.04s, 67.82MB read
wrk_test   | Requests/sec:  18931.54
wrk_test   | Transfer/sec:      2.71MB
wrk_test   | Round 1 |  Sandman:  Requests/sec:  27169.62  Req/Sec         Round 1 |  Uvicorn:  Requests/sec:  18931.54  Req/Sec
```
