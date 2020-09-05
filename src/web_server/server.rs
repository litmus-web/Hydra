/// server.rs is part of the main web server, this handles all of hyper.rs' systems like
/// serving requests and interactions with workers, this does not include the websocket systems
/// which is in a separate file or the worker process manager itself.

use std::sync::atomic::{AtomicUsize, Ordering};
use std::collections::HashMap;
use std::io;
use std::convert::Infallible;

use futures_util::future::join;

use hyper::server::Server;
use hyper::service::{make_service_fn, service_fn};
use hyper::{self, Body, Request, Response, StatusCode};
use hyper::http::Error;

use tokio::time::{delay_for, Duration};
use tokio::net::TcpListener;

use tokio_tungstenite::{tungstenite::protocol};


// Local references
use crate::web_server::websocket::handle_ws_connection;
use crate::web_server::structs::*;


/// A atomic counter used for the request ids, although this isn't strictly
/// needed because the system is single threaded per worker.
///
/// ```
/// // atomically increment the request id by 1.
/// let req_id = NEXT_USER_ID.fetch_add(1, Ordering::Relaxed);
/// ```
static NEXT_REQ_ID: AtomicUsize = AtomicUsize::new(1);


/// The overall server config struct, all worker threads should reference the
/// same struct and bind with SO_REUSEADDR if on unix systems or start the server
/// normally if on windows.
///
/// ```
/// let server_config = Config{
///     addr: String::from("127.0.0.1"),
///     port: 8080
/// };
/// ```
///
#[derive(Clone)]
pub struct Config {
    pub addr: String,
    pub port: u16,
}


/// The main entry point for the server, this starts the internal
/// runners and creates all the locks, when the server exits it will
/// log the error if applicable.
///
/// - free_port: Should be unique for every worker
/// - server_config: Should be the same for every worker to bind.
///
/// **Example:**
/// ```
/// let free_port: usize = 1234;
/// let server_config = Config{
///     addr: String::from("127.0.0.1"),
///     port: 8080
/// };
/// server::run(free_port, server_config).await;
/// ```
///
pub async fn run(free_port: usize, server_config: Config)  {
    let err = run_server(free_port, server_config).await;
    eprintln!("{:?}", err)
}

/// The internal server runner, this creates the workers and cache types
/// as well as creating the worker ws server and main server that then binds with
/// SO_REUSEADDR if on unix.
///
/// **Creates**
/// - Workers::default()
/// - Cache::default()
///
/// ```
/// // Server binds attached with SO_REUSEADDR.
/// let main_listener = TcpListener::bind(
///     format!("{}:{}", server_config.addr, server_config.port)).await?;
///
/// let worker_listener = TcpListener::bind(
///     format!("127.0.0.1:{}", free_port)).await?;
///
/// // Generating server builds and executors
/// let server_1 = Server::builder(hyper::server::accept::from_stream(main_listener))
///                     .executor(LocalExec)
///                     .http1_pipeline_flush(true)
///                     .serve(main_service);
///
/// let server_2 = Server::builder(hyper::server::accept::from_stream(worker_listener))
///                     .executor(LocalExec)
///                     .http1_pipeline_flush(true)
///                     .serve(worker_service);
///
///  // Join and await the two servers to start them running.
///  let _ = join(server_1, server_2).await;
/// ```
async fn run_server(free_port: usize, server_config: Config) -> io::Result<()> {
    // Generating ref cells to allow us to move data around.
    let workers = Workers::default();
    let cache = Cache::default();

    let worker_lock = workers.clone();
    let cache_lock = cache.clone();
    
    // Server binds attached with SO_REUSEADDR.
    let main_listener = TcpListener::bind(format!("{}:{}", server_config.addr, server_config.port)).await?;
    let worker_listener = TcpListener::bind(format!("127.0.0.1:{}", free_port)).await?;

    // Server service functions
    let main_service = make_service_fn(move |_| {
        let main_workers_clone = worker_lock.clone();
        let main_cache_clone = cache_lock.clone();

        async {
            Ok::<_, Infallible>(service_fn(move |req| {
                handle_incoming(req, main_workers_clone.clone(), main_cache_clone.clone())
            })) 
        }
    });

    let worker_service = make_service_fn(move |_| {
        let worker_workers_clone = workers.clone();
        let worker_cache_clone = cache.clone();

        async {
            Ok::<_, Infallible>(service_fn(move |req| {
                handle_workers_incoming(req, worker_workers_clone.clone(), worker_cache_clone.clone())
            })) 
        }
    });

    
    // Generating server builds and executors
    let server_1 = Server::builder(hyper::server::accept::from_stream(main_listener))
        .executor(LocalExec)
        .http1_pipeline_flush(true)
        .serve(main_service);

    let server_2 = Server::builder(hyper::server::accept::from_stream(worker_listener))
        .executor(LocalExec)
        .http1_pipeline_flush(true)
        .serve(worker_service);

    let _ = join(server_1, server_2).await;

    Ok(())
}

/// This handles *all* requests coming to the server binding on the user defined port.
/// This is ran as a local task with Tokio via Hyper's executor.
///
/// This get passed a reference pointer of Workers and Cache, along with the request itself
/// and returns a Result with the response or a error that hyper then handles.
///
/// **Logic Steps**
/// - Request parsed to a `OutgoingRequest`
/// - `OutgoingRequest` send to the ws shard
/// - waits for response from ws
/// - parses `ASGIResponse` to a `Response<body>` object.
///
/// **Invokes**
/// - `send_to_ws` (coro)
/// - `get_outgoing`
/// - `wait_for_response` (coro)
///
async fn handle_incoming(req: Request<Body>, workers: Workers, cache: Cache) -> Result<Response<Body>, Error> {
    // Lets take our shard id
    let shard = String::from("1");

    // If the shard id we got doesnt exist lets return with a inactive worker status,
    // just to be safe that we dont run into any errors.  
    if !workers.borrow().get(&shard).is_some() {
        return Ok(
            Response::builder()
                .status(StatusCode::from_u16(503).unwrap())
                .body("No workers active".into())
                .unwrap()
        );
    }

    // atomically increment the request id by 1.
    let req_id = NEXT_REQ_ID.fetch_add(1, Ordering::Relaxed);
   
    // Send it to the ws, in the end we dont really care about
    // the result at the moment because bigger issues will raise 
    // before it gets a chance.
    send_to_ws(
            &shard,
            workers,
            get_outgoing(req, req_id.clone())
        ).await;  
    
    // This will block/suspend the task until we get a request back or
    // it times out which will return a error instead.
    let asgi_resp = wait_for_response(&req_id, &cache).await;

    if asgi_resp.is_some() {
        return Ok(build_response(asgi_resp.unwrap()).unwrap())
    } 
    eprintln!("Server took too long to respond, Req Id: {}", req_id);

    let resp = Response::builder()
                    .status(StatusCode::from_u16(503).unwrap())
                    .body("Server took too long to respond.".into()).unwrap();              
    return Ok(resp)    
}

/// turns a `hyper::Request` into a `OutGoingRequest`
///
/// **Example Result**
/// ```
/// OutGoingRequest {
///    op: 0,
///    request_id: 1,
///    method: String::from("GET"),
///    remote: String::from("127.0.0.1"),
///    path: String::from("/hello/world"),
///    headers: map{"xyz": "abc"},
///    version: String::from("HTTP/1.1"),
///    body: String::from(""),
///    query: String::from("?name=bob&age=27")
/// }
/// ```
fn get_outgoing(mut req: Request<Body>, req_id: usize) -> OutGoingRequest {
    let headers: HashMap<String, String> = req
        .headers_mut()
        .drain()
        .map(|part| {(
                 String::from(part.0.unwrap().as_str()),
                 String::from(part.1.to_str().unwrap_or(""))
             )})
        .collect();

    OutGoingRequest {
        op: 0,
        request_id: req_id,
        method: String::from(req.method().as_str()),
        remote: String::from(req.uri().host().unwrap_or("127.0.0.1")),
        path: req.uri().path().to_string(),
        headers: headers,
        version: String::from("HTTP/1.1"),
        body: String::from(""),
        query: String::from(req.uri().query().unwrap_or(""))
    }
}

/// Sends a `OutGoingRequest` to the WS to the specified shard id.
/// - It specifically ignores the result overall, though this should be replaced
/// with a check later on.
///
async fn send_to_ws(shard_id: &String, workers: Workers, outgoing: OutGoingRequest) {
    let _ = workers
                .borrow()
                .get(shard_id)
                .unwrap()
                .send(
                    Ok(
                        protocol::Message::Text(
                            serde_json::to_string(&outgoing).unwrap()
                        )
                    )
                );
}

/// Specifically waits for the worker to send back a response and have it added to the cache
/// pool within 10ms otherwise the task is cancelled.
///
/// Returns: `Option<ASGIResponse>`
///
async fn wait_for_response(req_id: &usize, cache: &Cache) -> Option<ASGIResponse> {
    let mut time_out_count: u16 = 0;
    loop {
        if cache.borrow().get(&req_id).is_some() { 
            let res = cache.borrow_mut().remove(&req_id);
            return if res.is_some() { res } else { None }
        }
        delay_for(Duration::from_micros(25)).await;
        if time_out_count >= 400 {
            return None
        }
        else { time_out_count += 1 }
    }
}

/// Turns a `ASGIResponse` into a `hyper::Response<hyper::Body>` struct or error.
fn build_response(asgi_resp: ASGIResponse) -> Result<Response<Body>, Error> {    
    let mut resp = Response::builder()
                        .status(StatusCode::from_u16(asgi_resp.status).unwrap());
    
    for v in asgi_resp.headers.clone().iter() {
        resp = resp.header(v[0].as_str(), v[1].as_str()) 
    }
    resp.body(asgi_resp.clone().body.into())
}

/// Handles any worker connections, anything trying to connect that isn't connecting to
/// `ws://127.0.0.1/workers` will be returned 404 response code or another response
/// dependant on the WS upgrade being successful or not.
///
async fn handle_workers_incoming(
    req: Request<Body>,
    workers: Workers,
    cache: Cache
    ) -> Result<Response<Body>, Error> {
    match req.uri().path() {
        "/workers" => Ok(
            handle_worker(req, workers, cache).await
        ),
        _ => Ok(
                Response::builder()
                    .status(StatusCode::NOT_FOUND)
                    .body("Not Found".into())
                    .unwrap()
        )        
    }
}

/// Any incoming request for a worker is sent to a function to be upgraded and the response
/// returned back by `unwrap()`
///
async fn handle_worker(req: Request<Body>, workers: Workers, cache: Cache) -> Response<Body> {
    let w = handle_ws_connection(req, workers, cache).await;
    w.unwrap()
}

/// Local executor used for hyper to run, to maintain a single threaded
/// server and allow us to not need a lock for every request improving
/// performance especially when balancing with Python.
/// <br><br>
/// **Under the hood:**
/// ```
/// fn execute(&self, fut: F) {
///     tokio::task::spawn_local(fut);
/// }
/// ```
///
#[derive(Clone, Copy, Debug)]
struct LocalExec;

impl<F> hyper::rt::Executor<F> for LocalExec
where
    F: std::future::Future + 'static, // not requiring `Send`
{
    fn execute(&self, fut: F) {
        tokio::task::spawn_local(fut);
    }
}