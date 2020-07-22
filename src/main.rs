use tokio::io::{AsyncWriteExt, AsyncReadExt};
use tokio::sync::{mpsc, RwLock};

use futures::{FutureExt, StreamExt};

use warp::ws::{Message, WebSocket};
use warp::Filter;

use std::error::Error;
use std::sync::Arc;
use std::collections::HashMap;
use std::sync::atomic::{AtomicUsize, Ordering};

static NEXT_SYS_ID: AtomicUsize = AtomicUsize::new(1);

type WorkersSend = Arc<RwLock<HashMap<usize, mpsc::UnboundedSender<Result<Message, warp::Error>>>>>;
type Count = Arc<RwLock<usize>>;
type Cache = Arc<RwLock<HashMap<String, serde_json::Value>>>;


#[tokio::main(max_threads=1)]
async fn main() -> Result<(), Box<dyn Error>> {
    // Workers RwLock for sending channels
    let workers_lock_send = WorkersSend::default();
    let workers_clone_send = warp::any().map(move || workers_lock_send.clone());

    // Cache for storing responses from ws (because im lazy)
    let cache_lock = Cache::default();
    let cache_clone = warp::any().map(move || cache_lock.clone());

    // Workers RwLock for incremental counters TODO: Might be able to remove
    let counter_lock = Count::default();
    let _counter_clone = warp::any().map(move || counter_lock.clone());

    let workers = warp::path("workers")
        // The `ws()` filter will prepare Websocket handshake...
        .and(warp::ws())    // warp ws
        .and(workers_clone_send.clone())    // Workers send
        .and(cache_clone.clone())    // Cache store
        .map(|ws: warp::ws::Ws, wrk_send, cache| {
            ws.on_upgrade(
                move |socket| worker_connected(socket, wrk_send, cache)
            )
        });


    // Note: This should be moved to any endpoint and we just,
    // handle all endpoints with the same func and leave the rest
    // to the ASGI.
    let main_area = warp::path::full()         // Socket addr
        .and(workers_clone_send.clone())    // Workers send
        .and(cache_clone.clone())         // counter
        .and_then(any_path);

    println!("====================================================");
    println!("❤️  Sandman server running on http://127.0.0.1:8080.");
    println!("====================================================");

    let routes = workers.or(main_area);     // join main area and ws to the server

    warp::serve(routes).run(([0, 0, 0, 0], 8080)).await;
    Ok(())
}

async fn worker_connected(ws: WebSocket, wrk_send: WorkersSend, cache: Cache) {
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

    while let Some(result) = worker_ws_rx.next().await {
        let msg = match result {
            Ok(msg) => msg,
            Err(e) => {
                eprintln!("websocket connection cloded, reason: {:?}", e);
                break;
            }
        };
        let msg = if let Ok(s) = msg.to_str() {
            s
        } else {
            return;
        };
        let mut data: serde_json::Value = serde_json::from_str(msg).unwrap();
        let key: serde_json::Value = data["id"].take();
        let key = key.as_str().unwrap();
        cache.write().await.insert(key.to_owned(), data);
    }
}

async fn any_path(_path_to_serve: warp::path::FullPath, workers: WorkersSend, cache: Cache) -> Result<impl warp::Reply, warp::Rejection> {
    let content  = get_resp(workers, cache).await;
    match content {
        Ok(val) => Ok(warp::reply::html(val)),
        _ =>  {
            Ok(warp::reply::html("Oops! and error has ucoured.".to_owned()))
        }
    }    
}

async fn get_resp(workers: WorkersSend, _cache: Cache) -> Result<String, Box<dyn Error>> {
    
    let sys_id = NEXT_SYS_ID.fetch_add(1, Ordering::Relaxed);
    let part_1 = "{\"key\": ";
    let part_2 = format!("{}", sys_id);
    let part_3 = "}";
    let content = format!("{}{}{}", part_1, part_2, part_3);

    for (&_, tx) in workers.read().await.iter() {
        if let Err(_disconnected) = tx.send(Ok(Message::text(content.clone()))) {
        }
    }
    Ok(String::from("owo"))   
}