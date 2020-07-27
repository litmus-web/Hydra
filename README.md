

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

*Benchmarks have been removed after being invalid due to my containers not being setup right*
