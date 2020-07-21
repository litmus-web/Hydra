# WARNING! This repo is mearly a test and prototype of a system and is not complete nor production safe.

[![CodeFactor](https://www.codefactor.io/repository/github/project-dream-weaver/sandman/badge/master)](https://www.codefactor.io/repository/github/project-dream-weaver/sandman/overview/master)
# Sandman
A experemental Rust to Python hybrid webserver, combining the power of rust and the niceties of python.

The overall aim of this system is to allow compiled, production grade webservers to bind to python workers and frameworks that mearly do lightweight tasks rather than handling http requests directly. This allows for lower latency and few dropped requests due to Rust being able to cope with more.

Please note the current configuration has the following setup and is purely a proof of concept done in my free time.

**Rust Webserver (Warp/Hyper)** --> **Python Workers (Custom sockets)** --> **ASGI Frameworks / Binders**

## Currently Working
[x] - Basic implementation of the Tokio runtime and Warp http<br>
[x] - Basic load balancing betweeen python workers (I mean *very* basic)<br>
[x] - Serving sucessful http requests<br>

## Some of the todo list
[ ] - Adding proper asgi compatability (or custom system if asgi not possible)<br>
[ ] - Adding route construction <br>
[ ] - Better / actual query caching<br>
[ ] - Fix tokio blocking with RwLock<br>
[ ] - Static file serving<br>
[ ] - Implementing a query system between sockets and workers<br>
[ ] - Organising incoming worker request (Current system accepts each req as first in first out, which isnt true)<br>

### That Being said nothing like some benchmarks
**Please Note:**
- Take these benchmarks as a grain of salt, I cannot fully push these two systems to their limits before bottlenecks either the framework or wrk
- These benches were Sandman Vs Uvicorn (The current top benchmarked framework)
- Both systems were ran with single workers (Because my PC would top out otherwise) in a docker container with 2 Cores each and 4GB ram
- Both frameworks were tasked with serving a 10KB HTML index file, as personally i dont think a `Hello world` test means anything in the real world where these servers need to serve actual data not just two words.

Benchmarker used was [wrk](https://github.com/wg/wrk) 
- 2 Threads
- 400 connections
- 30 seconds of work

```
============ Sandman ============
web_1  | Running 30s test @ http://127.0.0.1:8080/bob
web_1  |   2 threads and 400 connections
web_1  |   Thread Stats   Avg      Stdev     Max   +/- Stdev
web_1  |     Latency   178.93ms   82.78ms   1.22s    93.71%
web_1  |     Req/Sec     1.14k   269.76     2.62k    72.08%
web_1  |   67249 requests in 30.05s, 569.60MB read
web_1  | Requests/sec:   2238.05
web_1  | Transfer/sec:     18.96MB

============ Uvicorn ============
web_1  | Running 30s test @ http://127.0.0.1:5050/bob
web_1  |   2 threads and 400 connections
web_1  |   Thread Stats   Avg      Stdev     Max   +/- Stdev
web_1  |     Latency   185.26ms  160.09ms   2.00s    97.11%
web_1  |     Req/Sec     1.20k   206.30     1.58k    79.83%
web_1  |   70207 requests in 30.04s, 595.82MB read
web_1  |   Socket errors: connect 0, read 0, write 0, timeout 87
web_1  | Requests/sec:   2337.10
web_1  | Transfer/sec:     19.83MB
```

An interesting note about these benchmarks is that altho Uvicorn wins by requests/sec Sandman is much more stable with handling many connections at once with 0 timeouts or errors compared to Uvicorns's 87 timeouts and higher avg and max latency

Uvicorn also uses the Uvloop event protocol while Sandman is running on pure asyncio Python *not* Uvloop which would also benifit Sandman.
