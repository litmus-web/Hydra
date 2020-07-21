use tokio::net::{TcpListener, TcpStream};
use tokio::io::{AsyncWriteExt, AsyncReadExt};
use tokio::sync::{mpsc, RwLock};

use futures::{FutureExt, StreamExt};

use warp::ws::{Message, WebSocket};
use warp::Filter;

use std::error::Error;
use std::sync::Arc;
use std::collections::HashMap;

type WorkersSend = Arc<RwLock<HashMap<usize, mpsc::UnboundedSender<Result<Message, warp::Error>>>>>;
type WorkersRecv = Arc<RwLock<HashMap<usize, mpsc::UnboundedReceiver<Result<Message, warp::Error>>>>>;

type Count = Arc<RwLock<usize>>;
type Cache = Arc<RwLock<HashMap<String, String>>>;


#[tokio::main]
async fn main() -> Result<(), Box<dyn Error>> {
    let addr = "127.0.0.1:9090";
    let mut listener = TcpListener::bind(addr).await.unwrap();

    // Workers RwLock for sending channels
    let workers_lock_send = WorkersSend::default();
    let workers_clone_send = warp::any().map(move || workers_lock_send.clone());

    // Workers RwLock for recv channels
    let workers_lock_recv = WorkersRecv::default();
    let workers_clone_recv = warp::any().map(move || workers_lock_recv.clone());

    // Workers RwLock for incremental counters TODO: Might be able to remove
    let counter_lock = Count::default();
    let counter_clone = warp::any().map(move || counter_lock.clone());

    // Experimental cache lock, this shouldnt really be used in prod.
    let cache_lock = Cache::default();
    let cache_clone = warp::any().map(move || cache_lock.clone());

    let workers = warp::path("workers")
        // The `ws()` filter will prepare Websocket handshake...
        .and(warp::ws())    // warp ws
        .and(workers_clone_send.clone())    // Workers send
        .and(workers_clone_recv.clone())    // Workers recv
        .map(|ws: warp::ws::Ws, wrk_send, wrk_recv| {
            ws.on_upgrade(
                move |socket| worker_connected(socket, wrk_send, wrk_recv)
            )
        });


    // Note: This should be moved to any endpoint and we just,
    // handle all endpoints with the same func and leave the rest
    // to the ASGI.
    let main_area = warp::path::end()
        .and(warp::addr::remote())          // Socket addr
        .and(workers_clone_send.clone())    // Workers send
        .and(workers_clone_recv.clone())    // Workers recv
        .and(counter_clone.clone())         // counter
        .and_then(all_routes);

    println!("====================================================");
    println!("❤️  Sandman server running on http://127.0.0.1:8080.");
    println!("====================================================");

    let routes = main_area.or(workers);     // join main area and ws to the server

    warp::serve(routes).run(([0, 0, 0, 0], 8080)).await;
    Ok(())
}

async fn worker_connected(ws: WebSocket, wrk_send: WorkersSend, wrk_recv: WorkersRecv) {
    println!("🦄  Accepted connection from worker");

    // split the ws so we can forward the transmissions
    let (worker_ws_tx, mut worker_ws_rx) = ws.split();

    // Unbound channel for forwarding messages to ws
    let (tx, rx) = mpsc::unbounded_channel();
    tokio::task::spawn(rx.forward(worker_ws_tx).map(|result| {
        if let Err(e) = result {
            eprintln!("websocket send error: {}", e);
        }
    }));

    wrk_send.write().await.insert(1, tx);
    let wrk_send_2 = wrk_send.clone();

    while let Some(result) = worker_ws_rx.next().await {
        let msg = match result {
            Ok(msg) => msg,
            Err(e) => {
                eprintln!("websocket error {:?}", e);
                break;
            }
        };
        println!("{:?}", msg)
    }
}


async fn all_routes(agent: Option<std::net::SocketAddr>, workers_send: WorkersSend, workers_recv: WorkersRecv, counter: Count) -> Result<impl warp::Reply, warp::Rejection> {
    let agent = format!("{:?}", agent);

    let content  = get_resp().await;

    match content {
        Ok(val) => Ok(warp::reply::html(val)),
        _ =>  Ok(warp::reply::html("Oops! and error has ucoured.".to_owned()))
    }    
}

async fn get_resp() -> Result<String, Box<dyn Error>> { 
           
        Ok(String::from("s: &str"))  
}
