use std::sync::Arc;
use std::collections::HashMap;
use std::cell::Cell;
use std::rc::Rc;
use std::io;
use std::convert::Infallible;
use std::str;

use futures::prelude::*;
use futures::stream::StreamExt;

use headers::{self, HeaderMapExt};

use hyper::header::{self, AsHeaderName, HeaderMap, HeaderValue};
use hyper::server::Server;
use hyper::service::{make_service_fn, service_fn};
use hyper::{self, Body, Method, Request, Response, StatusCode};

use tokio::sync::RwLock;
use tokio::net::TcpListener;

use tokio_tungstenite::{tungstenite::protocol, WebSocketStream};

pub type Workers = Arc<RwLock<HashMap<String, String>>>;

#[derive(Clone)]
pub struct Config {
    pub addr: &'static str,
    pub port: u16,
}

pub async fn run(process_id: usize, server_config: Config, workers: Workers)  { 
    println!(
        "[ WORKER {} ] Worker starting...", process_id);
    
    let _ = run_server(process_id, server_config, workers).await;
}

async fn run_server(process_id: usize, server_config: Config, workers: Workers) -> io::Result<()> {

    let listener = TcpListener::bind(format!("{}:{}", server_config.addr, server_config.port)).await?;

    // Using a !Send request counter is fine on 1 thread...
    let counter: std::rc::Rc<std::cell::Cell<usize>> = Rc::new(Cell::new(0));

    let make_service = make_service_fn(move |_| {
        // For each connection, clone the counter to use in our service...
        let cnt = counter.clone();

        async {
            Ok::<_, Infallible>(service_fn(move |req| {
                handle_incomming(req)
            })) 
        }
    });
    
    let server = Server::builder(hyper::server::accept::from_stream(listener))
        .executor(LocalExec)
        .http1_pipeline_flush(true)
        .serve(make_service)
        .await;

    if let Err(err) = server {
        eprintln!("Error when starting {:?}", err)
    };
    Ok(())
}


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

async fn handle_incomming(req: Request<Body>) -> Result<Response<Body>, io::Error> {
    match (req.method(), req.uri().path()) {
        (&Method::GET, "/workers") => Ok(handle_worker(req).await),
        _ => Ok(handle_any(req).await),
    }
}

async fn handle_any(req: Request<Body>) -> Response<Body> {
    Response::builder()
        .status(StatusCode::NOT_FOUND)
        .body("Not Found".into())
        .unwrap()
}

async fn handle_worker(req: Request<Body>) -> Response<Body> {
    let w = handle_ws_connection(req).await;
    w.unwrap()
}

async fn handle_ws_connection(req: Request<Body>) -> Result<Response<Body>, io::Error> {
    let res = match upgrade_connection(req).await {
        Err(res) => res,
        Ok((res, ws)) => {

            // Create a very basic async task for a echo ws test.
            let run_ws_task = async {
                match ws.await {
                    Ok(ws) => {
                        
                        // Dud counter just for testing
                        let mut counter: usize = 0;

                        // Split the ws connection into sender and reciever...
                        let (tx, rc) = ws.split();

                        // Check if the future is Ok, map that then
                        // to a new response formatting with the message.
                        let rc = rc.try_filter_map(|m| {
                            future::ok(match m {
                                protocol::Message::Text(text) => {
                                    counter += 1;
                                    Some(protocol::Message::text(format!(
                                        "Response {}: {}",
                                        counter, text
                                    )))
                                }
                                _ => None,
                            })
                        });

                        // Try forward the message onto the sender
                        match rc.forward(tx).await {
                            Err(e) => eprintln!("WS Error {}", e),
                            Ok(_) => println!("Websocket has ended"),
                        }
                    }
                    Err(_e) => eprintln!("WS error"),
                }
            };
            tokio::spawn(run_ws_task);
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
            let r = WebSocketStream::from_raw_socket(upgraded, protocol::Role::Server, None).await;
            Ok(r)
        });

    Ok((res, upgraded))
}