# WARNING! This repo is mearly a test and prototype of a system and is not complete nor production safe.

[![CodeFactor](https://www.codefactor.io/repository/github/project-dream-weaver/sandman/badge/master)](https://www.codefactor.io/repository/github/project-dream-weaver/sandman/overview/master)
# Sandman
A experemental Rust to Python hybrid webserver, combining the power of rust and the niceties of python.

The overall aim of this system is to allow compiled, production grade webservers to bind to python workers and frameworks that mearly do lightweight tasks rather than handling http requests directly. This allows for lower latency and few dropped requests due to Rust being able to cope with more.

Please note the current configuration has the following setup and is purely a proof of concept done in my free time.

**Rust Webserver (Warp/Hyper)** --> **Python Workers (Custom sockets)** --> **ASGI Frameworks / Binders**

## Currently Working
[x] - Basic implementation of the Tokio runtime and Warp http<br>
[x] - Serving sucessful http requests<br>

## Some of the todo list
[ ] - Adding proper asgi compatability (or custom system if asgi not possible)<br>
[ ] - Adding route construction <br>
[ ] - Better / actual query caching<br>
[x*] - Fix tokio blocking with RwLock - Sorta? fixed?<br>
[ ] - Static file serving<br>
[ ] - Implementing a query system between sockets and workers<br>
[x] - Organising incoming worker request (Current system accepts each req as first in first out, which isnt true)<br>

### That Being said nothing like some benchmarks -- UPDATED
**Please Note:**
- Take these benchmarks as a grain of salt, I cannot fully push these two systems to their limits before bottlenecks either the framework or wrk
- These benches were Sandman Vs Uvicorn (The current top benchmarked framework)
- Both systems were ran with single workers (Because my PC would top out otherwise) in a docker container with 2 Cores each and 4GB ram
- Both frameworks were tasked with serving a 2KB HTML fortunes file, as personally i dont think a `Hello world` test means anything in the real world where these servers need to serve actual data not just two words.

Benchmarker used was [wrk](https://github.com/wg/wrk) 
- 2 Threads
- 512 connections
- 30 seconds of work

```docker
============= Sandman =============
web_1  | Running 30s test @ http://127.0.0.1:8080/
web_1  |   2 threads and 512 connections
web_1  |   Thread Stats   Avg      Stdev     Max   +/- Stdev
web_1  |     Latency    55.65ms   32.41ms 610.55ms   93.26%
web_1  |     Req/Sec     4.80k     0.95k    6.42k    88.93%
web_1  |   284886 requests in 30.05s, 365.42MB read
web_1  | Requests/sec:   9481.20
web_1  | Transfer/sec:     12.16MB

============= Uvicorn =============
web_1  | Running 30s test @ http://127.0.0.1:5050/
web_1  |   2 threads and 512 connections
web_1  |   Thread Stats   Avg      Stdev     Max   +/- Stdev
web_1  |     Latency    86.21ms  111.47ms   1.94s    92.57%
web_1  |     Req/Sec     3.92k     1.40k    5.89k    81.54%
web_1  |   232351 requests in 30.06s, 302.47MB read
web_1  |   Socket errors: connect 0, read 14, write 0, timeout 2
web_1  | Requests/sec:   7730.09
web_1  | Transfer/sec:     10.06MB
```
**Updated:** 
Sandman has starter to go *quick* as you can see in this demonstration not only do we have a lower latency between requests but we also have 0 socket errors and 1,700 more req a sec. In this instance Uvicorn is deployed with Gunicorn. Sandman is set to max_threads=1 why? I have no idea, it seems to just go faster limited. If someone could tell me why id appriciate it.

Uvicorn also uses the Uvloop event protocol while Sandman is running on pure asyncio Python *not* Uvloop which would also benifit Sandman.
