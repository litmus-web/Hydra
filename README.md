# WARNING! This repo is mearly a test and prototype of a system and is not complete nor production safe.

[![CodeFactor](https://www.codefactor.io/repository/github/project-dream-weaver/sandman/badge/master)](https://www.codefactor.io/repository/github/project-dream-weaver/sandman/overview/master)
# Sandman
A experemental ~~Rust~~ Golang to Python hybrid webserver, combining the power of Golang and the niceties of python.

The overall aim of this system is to allow compiled, production grade webservers to bind to python workers and frameworks that mearly do lightweight tasks rather than handling http requests directly.

Please note the current configuration has the following setup and is purely a proof of concept done in my free time.

## We've moved to Golang!
Why? Originally we were using Rust and Warp which is built off the hyper http crate however, we ran into issues with locks due to send methods requiring a mutable lock, because i am not experienced anough in rust to fix that issue at this time we've witched to Golang which provides a very Fast http client and makes life a bit easier for me to develop. :D


## New Benchmarks
Please take this benchmakrs mearly as proof of what these sorts of system can achieve rather than me saying "Look you should use this" as I cannot push either systems to their limit with my current setup.

**Benchmark setup**
Using [wrk](https://github.com/wg/wrk)
__Settings__
- 2 threads
- 256 concurrency
- 30s work


```
benchmark_1       | Round 1 |  Sandman:  Requests/sec:  28209.18  Req/Sec         Round 1 |  Uvicorn:  Requests/sec:  22603.35  Req/Sec
benchmark_1       | Round 2 |  Sandman:  Requests/sec:  28149.09  Req/Sec         Round 2 |  Uvicorn:  Requests/sec:  24908.73  Req/Sec
benchmark_1       | Round 3 |  Sandman:  Requests/sec:  28206.76  Req/Sec         Round 3 |  Uvicorn:  Requests/sec:  21675.43  Req/Sec
```
