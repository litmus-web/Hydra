

[![CodeFactor](https://www.codefactor.io/repository/github/project-dream-weaver/sandman/badge/master)](https://www.codefactor.io/repository/github/project-dream-weaver/sandman/overview/master)


# Sandman
A experemental Rust to Python hybrid webserver accelerating Python frameworks to warp speed.

### WARNING! This repo is mearly a test and prototype of a system and is not complete nor production safe.

The overall aim of this system is to allow compiled, production grade webservers to bind to python workers and frameworks that mearly do lightweight tasks rather than handling http requests directly. This allows for lower latency and few dropped requests due to Rust being able to cope higher load.

At its current state Sandman *is* faster (in my testing) than [Uvicorn](https://www.uvicorn.org/) (the current leader) however Sandman still lacks alot of features.<br>

Sandman makes use of the Rust framework [Warp](https://github.com/seanmonstar/warp) (built off [hyper.rs](https://hyper.rs/)), I could of chosen to use Actix-Web however, I chose not to due to Warp's minimalist system and development rate compared to Actix's drama.

Please note the current configuration has the following setup and is purely a proof of concept done in my free time.

**Rust Webserver (Warp/Hyper)** --> **Python Workers** --> **ASGI / WSGI Frameworks / Binders**

## Currently Working
[x] - Basic implementation of the Tokio runtime and Warp http.<br>
[x] - Serving sucessful http requests.<br>
[x] - Add scaling for higher loads.<br>
[x] - Add some loose WSGI workers to let me play with that.<br>

## Some of the todo list
[-] - Adding proper asgi compatability (or custom system if asgi not possible)<br>
[-] - Static file serving<br>
[-] - CLI commands<br>
[-] - Internal process manager not manual startup<br>

## Benchmarks

**Note:** 
- These benchmarks do not mean much other than to show the potential of Sandman,<br>
- I decided to take Uvicorn and Aiohttp as two rival frameworks so you can get an idea of how the diffrence in numbers can compare (Uvicorn and Aiohttp are both regularly benchmarked far better than I could ever do).<br>
- It is also important to note that none of these frameworks were truly pushed to their limits, they were just pushed as hard as my Pc can push them, Wrk was set to 2 threads and dockered to provide some method of control and regulation.
- I would *highly* advise taking a look at [Techempower's Benchmarks](https://www.techempower.com/benchmarks/#section=data-r19&hw=ph&test=fortune&l=zijzen-1r) for a comparrison of what these number mean (TE's servers are *alot* more powerful than my Pc mind you)

Benchmarker used was [wrk](https://github.com/wg/wrk) 
- 2 Threads
- 256 connections
- 15 seconds of work
ran 3 times the best output from each framework was selected.

```
============= Sandman =============
Running 15s test @ http://127.0.0.1:8080/
  2 threads and 256 connections
   Thread Stats   Avg      Stdev     Max   +/- Stdev
     Latency    24.51ms   17.70ms 321.11ms   96.77%
     Req/Sec     5.71k     0.89k    6.47k    94.22%
   167302 requests in 15.04s, 20.90MB read
  Requests/sec:  11123.87
  Transfer/sec:      1.39MB

============= Uvicorn =============
Running 15s test @ http://127.0.0.1:5050/
   2 threads and 256 connections
   Thread Stats   Avg      Stdev     Max   +/- Stdev
     Latency    35.59ms   29.73ms 307.55ms   91.56%
     Req/Sec     4.18k     1.43k    5.89k    83.56%
   124200 requests in 15.04s, 17.89MB read
 Requests/sec:   8259.34
 Transfer/sec:      1.19MB
 
============= AioHttp =============
Running 15s test @ http://127.0.0.1:8000/
   2 threads and 256 connections
   Thread Stats   Avg      Stdev     Max   +/- Stdev
     Latency    42.34ms   39.12ms 832.40ms   95.83%
     Req/Sec     3.43k   649.43     4.32k    91.28%
   101892 requests in 15.03s, 15.84MB read
 Requests/sec:   6777.97
 Transfer/sec:      1.05MB
```


Uvicorn also uses the Uvloop event protocol while Sandman is running on pure asyncio Python *not* Uvloop which would also benifit Sandman.
