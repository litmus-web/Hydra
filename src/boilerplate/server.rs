use std::sync::atomic::{AtomicUsize, Ordering};
use std::collections::HashMap;
use std::rc::Rc;
use std::cell::RefCell;
use std::io;
use std::convert::Infallible;
use std::str;

use futures::prelude::*;
use futures::{FutureExt, StreamExt};

use futures_util::future::join;

use headers::{self, HeaderMapExt};

use hyper::header::{self, AsHeaderName, HeaderMap, HeaderValue};
use hyper::server::Server;
use hyper::service::{make_service_fn, service_fn};
use hyper::{self, Body, Request, Response, StatusCode};
use hyper::http::Error;

use tokio::net::TcpListener;
use tokio::sync::mpsc;

use tokio_tungstenite::{tungstenite::protocol, WebSocketStream};
use tokio_tungstenite::tungstenite;

use serde_json;
use serde::{Serialize, Deserialize};

///
///  Static globals
///
static NEXT_USER_ID: AtomicUsize = AtomicUsize::new(1);

///
///  Custom types for workers, cache, etc...
/// 
///     - Workers: Set in a HashMap contained in a RC to allow us to use Default().
///     - Thread Safety: This is fine because the system is Single threaded.
type Workers = RefCell<HashMap<String, mpsc::UnboundedSender<Result<protocol::Message, tungstenite::error::Error>>>>;
type Cache = RefCell<HashMap<String, ASGIResponse>>;


///
///  General Structs for communication between systems.
///
#[derive(Clone)]
pub struct Config {
    pub addr: String,
    pub port: u16,
}

#[derive(Clone, Serialize, Deserialize)]
struct OutGoingRequest {
    request_id: usize,
    method: String,
    remote: String,
    path: String,
    headers: HashMap<String, String>,
}

#[derive(Clone, Serialize, Deserialize)]
struct ASGIResponse {
    request_id: String,
    status: u16,
    headers: HashMap<String, String>,
    body: String
}



///
///  Runners for main function.
///
pub async fn run(target_id: usize, server_config: Config)  {     
    let _ = run_server(target_id, server_config).await;
}

async fn run_server(target_id: usize, server_config: Config) -> io::Result<()> {
    // Generating ref cells to allow us to move data around.
    let workers = Workers::default();
    let cache = Cache::default();

    let worker_lock = workers.clone();
    let cache_lock = cache.clone();
    
    // Server binds, `?` makes the binder attach with SO_REUSEADDR, using unwrap() keeps the ports individual.
    let main_listener = TcpListener::bind(format!("{}:{}", server_config.addr, server_config.port)).await?;
    let worker_listener = TcpListener::bind(format!("{}:{}", server_config.addr, target_id)).await.unwrap();

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
    let req_id = NEXT_USER_ID.fetch_add(1, Ordering::Relaxed);

    let outgoing = OutGoingRequest {
        request_id: req_id,
        method: req.method().to_string(),
        remote: String::from(req.uri().host().unwrap()),
        path: req.uri().path().to_string(),
        headers: Default::default(),
    };

    let outgoing = serde_json::to_string(&outgoing).unwrap();

    let _ = workers
        .borrow()
        .get(&String::from("main"))
        .unwrap()
        .send(Ok(protocol::Message::Text(outgoing)));

    Ok(
        Response::builder()
        .status(StatusCode::NOT_FOUND)
        .body("Not Found".into())
        .unwrap()
    )
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
        "/workers" => Ok(handle_worker(req, workers, cache).await),
        _ => {
            Ok(
                Response::builder()
                    .status(StatusCode::NOT_FOUND)
                    .body("Not Found".into())
                    .unwrap()
            )
        }
    }
}

async fn handle_worker(req: Request<Body>, workers: Workers, cache: Cache) -> Response<Body> {
    let w = handle_ws_connection(req, workers, cache).await;
    w.unwrap()
}

async fn handle_ws_connection(
    req: Request<Body>,
    workers: Workers,
    cache: Cache,
    ) -> Result<Response<Body>, io::Error> {

    let res = match upgrade_connection(req).await {
        Err(res) => res,
        Ok((res, ws)) => {
            let run_ws_task = async {
                match ws.await {
                    Ok(ws) => {
                        // Split the ws connection into sender and reciever...
                        let (ws_tx, mut ws_rc) = ws.split();
                        let (tx, rx) = mpsc::unbounded_channel();

                        tokio::task::spawn_local(rx.forward(ws_tx).map(|result| {
                            if let Err(e) = result {
                                eprintln!("websocket send error: {}", e);
                            }
                        }));
                        
                        workers.borrow_mut().insert(String::from("main"), tx);

                        // Run it until something breaks or it stops normally.
                        while let Some(result) = ws_rc.next().await {
                            let msg = match result {
                                Ok(protocol::Message::Text(text)) => text,
                                Err(e) => {
                                    eprintln!("websocket error {}", e);
                                    break;
                                },
                                _ => String::from("None")
                            };

                            let outgoing: ASGIResponse = serde_json::from_str(msg.as_str()).unwrap();
                            let mut borred = cache.borrow_mut();
                            borred.insert(outgoing.clone().request_id, outgoing);

                        }
                    }
                    Err(_e) => eprintln!("WS error"),
                }
                (cache, workers)
            };
            tokio::task::spawn_local(run_ws_task);
            res
        }
    };
    Ok(res)
}

fn header_matches<S: AsHeaderName>(headers: &HeaderMap<HeaderValue>, name: S, value: &str) -> bool {
    headers
        .get(name)
        .and_then(|v| v.to_str().ok())
        .map(|v| v.to_lowercase() == value)
        .unwrap_or(false)
}

async fn upgrade_connection(req: Request<Body>) -> Result<(
        Response<Body>,
        impl Future<Output = Result<WebSocketStream<hyper::upgrade::Upgraded>, ()>> + Send,
    ), Response<Body>> {
    // Upgrades the Request with the 3 way handshake to a websocket.
    
    // Generate a new response to populate later
    let mut res = Response::new(Body::empty());
    let mut header_error = false;

    if !header_matches(req.headers(), header::UPGRADE, "websocket") {
        header_error = true;
    }
    
    if !header_matches(req.headers(), header::SEC_WEBSOCKET_VERSION, "13") {
        header_error = true;
    }

    if !req
        .headers()
        .typed_get::<headers::Connection>()
        .map(|h| h.contains("Upgrade"))
        .unwrap_or(false)
    {
        header_error = true;
    }

    let key = req.headers().typed_get::<headers::SecWebsocketKey>();

    if key.is_none() {
        header_error = true;
    }

    if header_error {
        *res.status_mut() = StatusCode::BAD_REQUEST;
        return Err(res);
    }

    *res.status_mut() = StatusCode::SWITCHING_PROTOCOLS;
    let h = res.headers_mut();
    h.typed_insert(headers::Upgrade::websocket());
    h.typed_insert(headers::SecWebsocketAccept::from(key.unwrap()));
    h.typed_insert(headers::Connection::upgrade());

    // If this succeeds the request has upgraded otherwise errors.
    let upgraded = req
        .into_body()
        .on_upgrade()
        .map_err(|err| eprintln!("Cannot create websocket: {} ", err))
        .and_then(|upgraded| async {
            let r = WebSocketStream::from_raw_socket(
                upgraded,
                protocol::Role::Server,
                None
            ).await;
            Ok(r)
        });

    Ok((res, upgraded))
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