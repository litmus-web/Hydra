use std::sync::atomic::{AtomicUsize, Ordering};
use std::collections::HashMap;
use std::io;
use std::convert::Infallible;
use std::str;

use futures::TryStreamExt; 

use futures_util::future::join;

use hyper::server::Server;
use hyper::service::{make_service_fn, service_fn};
use hyper::{self, Body, Request, Response, StatusCode};
use hyper::http::Error;

use tokio::time::{delay_for, Duration};
use tokio::net::TcpListener;

use tokio_tungstenite::{tungstenite::protocol};

use serde::{Serialize, Deserialize};


/// Local references
use crate::web_server::websocket::handle_ws_connection;
use crate::web_server::structs::*;


///
///  Static globals
///
static NEXT_USER_ID: AtomicUsize = AtomicUsize::new(1);


///
///  General Structs for communication between systems.
///
#[derive(Clone)]
pub struct Config {
    pub addr: String,
    pub port: u16,
}


///
///  Runners for main function.
///
pub async fn run(free_port: usize, server_config: Config)  {
    let err = run_server(free_port, server_config).await;
    eprintln!("{:?}", err)
}

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



///
///  Main area where all incoming requests get sent.
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

    // atomically increment the request id
    let req_id = NEXT_USER_ID.fetch_add(1, Ordering::Relaxed);
   
    // Send it to the ws
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

async fn wait_for_response(req_id: &usize, cache: &Cache) -> Option<ASGIResponse> {
    let mut time_out_count: u16 = 0;
    loop {
        if cache.borrow().get(&req_id).is_some() { 
            let res = cache.borrow_mut().remove(&req_id);
            if res.is_some() { return res }
            else { return None }
        }
        delay_for(Duration::from_micros(15)).await;
        if time_out_count >= 200 { 
            return None
        }
        else { time_out_count += 1 }
    }
}

fn build_response(asgi_resp: ASGIResponse) -> Result<Response<Body>, Error> {    
    let mut resp = Response::builder()
                        .status(StatusCode::from_u16(asgi_resp.status).unwrap());
    
    for v in asgi_resp.headers.clone().iter() {
        resp = resp.header(v[0].as_str(), v[1].as_str()) 
    }
    resp.body(asgi_resp.clone().body.into())
}



///
///  This is the worker area, responsible for upgrading the WS connection
///  to the server allowing for fast transactions between processes.
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

async fn handle_worker(req: Request<Body>, workers: Workers, cache: Cache) -> Response<Body> {
    let w = handle_ws_connection(req, workers, cache).await;
    w.unwrap()
}



///
///  Local future exectutor for hyper, cuz? Threadsafety ig.
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