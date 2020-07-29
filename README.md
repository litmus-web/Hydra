# WARNING! This repo is mearly a test and prototype of a system and is not complete nor production safe.

[![CodeFactor](https://www.codefactor.io/repository/github/project-dream-weaver/sandman/badge/master)](https://www.codefactor.io/repository/github/project-dream-weaver/sandman/overview/master)
# Sandman
A experemental Rust to Python hybrid webserver, combining the power of rust and the niceties of python.

## We've moved to Golang!
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

