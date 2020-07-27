use tokio::sync::{mpsc, RwLock};
use tokio::time::{delay_for, Duration};
use tokio::runtime::{Builder};
use tokio::net::TcpListener;
use tokio::process::Command;

use futures::{FutureExt, StreamExt};

use serde_json::json;
use serde::{Deserialize, Serialize};

use warp::ws::{Message, WebSocket};
use warp::{
    Filter,
    http::HeaderMap, 
    http::StatusCode, 
    http::Response,
};
use warp::hyper;

use std::sync::Arc;
use std::collections::HashMap;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::thread;
use std::net::Ipv4Addr;

use colored::*;

use chrono::{DateTime, Utc};

// use clap::{Arg, App};


static NEXT_SYS_ID: AtomicUsize = AtomicUsize::new(1);

type WorkersSend = Arc<RwLock<HashMap<usize, mpsc::UnboundedSender<Result<Message, warp::Error>>>>>;
type Cache = Arc<RwLock<HashMap<String, ASGIResponse>>>;


#[derive(Clone)]
struct WorkerConfig {
    id: usize,
    host: Ipv4Addr,
    port: u16,
    thread_count: usize,
}

#[derive(Serialize, Deserialize, Clone)]
struct ASGIResponse {
    id: String,
    body: String,
    status: u16,
    headers: Vec<Vec<String>>,
}


fn main() {    

    // some stuff
    let threads: usize = 3;

    let host: Ipv4Addr = Ipv4Addr::new(0, 0, 0, 0);
    let port: u16 = 8080;
    
    
    println!("{}", "=============== Starting Sandman =============".cyan());
    println!("  Worker count:  {}", threads);
    println!("  Worker path  @ ws://127.0.0.1:{}/workers", port);
    println!("  Server running  @ http://127.0.0.1:{}/", port);
    println!("{}", "==============================================".cyan());
    

    for i in 1..threads {
        let config = WorkerConfig{
            id: i,
            host: host.clone(),
            port,
            thread_count: threads,
        };
        // Setting some things before we start
        let target_path = String::from("ASGI_Raw:app"); 
        let temp = target_path.split(":").collect::<Vec<&str>>();
        let file_path = temp[0];

        let _child = Command::new("python")
            .arg(format!("{}.py", file_path))
            .arg(format!("{}", config.id))
            .spawn();

        let _ = thread::Builder::new().name(
            format!("sandman-worker-{}", i).to_string()).spawn(
                move || { start_workers(config) }
            );
    };    

    

    let config = WorkerConfig{
            id: threads,
            host: host.clone(),
            port,
            thread_count: threads,
    };

    // Setting some things before we start
    let target_path = String::from("ASGI_Raw:app"); 
    let temp = target_path.split(":").collect::<Vec<&str>>();
    let file_path = temp[0];

    let _child = Command::new("python")
        .arg(format!("{}.py", file_path))
        .arg(format!("{}", config.id))
        .spawn();

    start_workers(config)
}

fn start_workers(cfg: WorkerConfig) {
    let mut rt = Builder::new()
    .basic_scheduler()
    .enable_all()
    .build()
    .unwrap();

    let _ = rt.block_on(run_main(cfg));
}

async fn run_main(cfg: WorkerConfig) -> Result<(), Box<dyn std::error::Error>> {   

    
    // Logging
    log_info(format!("{} {}", "[ Rust Worker ]".red(), "Starting thread."));


    // RwLocks
    let workers_lock_send = WorkersSend::default();
    let workers_clone_send = warp::any().map(move || workers_lock_send.clone());

    let cache_lock = Cache::default();
    let cache_clone = warp::any().map(move || cache_lock.clone());

    let cfg_lock = cfg.clone();
    let cfg_clone = warp::any().map(move || cfg_lock.clone());
 

    //  Endpoints 
    let workers = warp::path("workers")
        // The `ws()` filter will prepare Websocket handshake...
        .and(warp::ws())                    // Warp ws
        .and(cfg_clone.clone())             // Config copy
        .and(workers_clone_send.clone())    // Workers send
        .and(cache_clone.clone())           // Cache store
        .map(|ws: warp::ws::Ws, cfg, wrk_send, cache| {
            ws.on_upgrade(
                // Call worker connected if the WS is upgraded
                move |socket| worker_connected(socket, cfg, wrk_send, cache)
            )
        });

    let main_area = warp::path::full()          // Any path will hit this
        .and(cfg_clone.clone())                 // Config copy
        .and(warp::header::headers_cloned())    // Headers
        .and(warp::method())                    // Method
        .and(workers_clone_send.clone())        // Workers send
        .and(cache_clone.clone())               // counter
        .and_then(any_path);

    let routes = workers.or(main_area);     // join main area and ws to the server  

 
    //  Webserver Binding 
    let listener = TcpListener::bind((cfg.host, cfg.port)).await?;
    warp::serve(routes).run_incoming(listener).await;
    Ok(())
}


//  Worker Area
async fn worker_connected(ws: WebSocket, cfg: WorkerConfig, wrk_send: WorkersSend, cache: Cache) {    

    // split the ws so we can forward the transmissions
    let (worker_ws_tx, mut worker_ws_rx) = ws.split();

    // Unbound channel for forwarding messages to ws
    let (tx, rx) = mpsc::unbounded_channel();
    tokio::task::spawn(rx.forward(worker_ws_tx).map(|result| {
        if let Err(e) = result {
            eprintln!("websocket send error: {}", e);
        }
    }));

    if let Err(_disconnected) = tx.send(Ok(Message::text("HELLO.WORLD"))) {
            eprintln!("Worker has lost connection to reciever, restart Sandman.") 
    }
    
    let mut key = serde_json::Value::default();
    while let Some(result) = worker_ws_rx.next().await {
        let msg = match result {
            Ok(msg) => msg,
            Err(e) => {
                wrk_send.write().await.remove(&cfg.id);
                // Logging
                log_info(
                    format!("{} {}", "[ ERROR ]".red(),
                     format!("websocket connection cloded, reason: {:?}", e).red()
                    )
                );
                break;
            }
        };
        let msg = if let Ok(s) = msg.to_str() {
            s
        } else {
            return;
        };
        let mut data: serde_json::Value = serde_json::from_str(msg).unwrap();
        key = data["worker_id"].take();
        break
    }
    wrk_send.write().await.insert(key.as_str().unwrap_or("0").parse().unwrap_or(0), tx);

    // Logging
    log_info(
        format!("{} {}", "[ Python Worker ]".blue(), "Connected to WS.")
    );

    while let Some(result) = worker_ws_rx.next().await {
        let msg = match result {
            Ok(msg) => msg,
            Err(e) => {
                wrk_send.write().await.remove(&cfg.id);
                log_info(
                    format!("{} {}", "[ ERROR ]".red(),
                     format!("websocket connection cloded, reason: {:?}", e).red()
                    )
                );
                break;
            }
        };

        let msg = if let Ok(s) = msg.to_str() {
            s
        } else {
            return;
        };
        let data: ASGIResponse = serde_json::from_str(msg).unwrap();
        cache.write().await.insert(data.id.clone(), data);
    }
}



//  Endpoint area
async fn any_path(
    path_to_serve: warp::path::FullPath, 
    cfg: WorkerConfig, 
    headers: HeaderMap, 
    http_method: hyper::Method,
    workers: WorkersSend, 
    cache: Cache
    ) -> Result<impl warp::Reply, warp::Rejection> {

    if workers.read().await.len() < cfg.thread_count {
        Ok(
            Response::builder()
                .status(503)
                .body(String::from("The server has not finished starting!"))
                .unwrap()
        )
    } else {
        let response  = get_resp(
            path_to_serve,
            cfg, 
            headers, 
            http_method, 
            workers, 
            cache).await;

        match response {
            Ok(r) => Ok(r),
            _ => Ok(
                Response::builder()
                    .status(503)
                    .body(String::from("The server has ran into an error."))
                    .unwrap()
            ),
        }
        
    }    
}


async fn get_resp(
    path_to_serve: warp::path::FullPath, 
    cfg: WorkerConfig, 
    headers: HeaderMap, 
    http_method: hyper::Method,
    workers: WorkersSend, 
    cache: Cache
    ) -> warp::http::Result<Response<String>> {

    let sys_id = NEXT_SYS_ID.fetch_add(1, Ordering::Relaxed);
    let mut header_map = HashMap::new();

    for (key, value) in headers.iter() {
        header_map.insert(
            format!("{}", key.as_str()),
             format!("{}", value.to_str().unwrap_or("None"))
            );
    }

    let mut id_: usize = 1;
    if sys_id % 2 == 0 {
        id_ = 2;
    } else if sys_id % 3 == 0 {
        id_ = 3;
    } 

    let sys_id = sys_id.to_string();

    let query = json!({
        "id": sys_id,
        "context": {
            "headers": header_map,
            "path": path_to_serve.as_str(),
            "method": http_method.as_str(),
            "port": cfg.port.to_string(),
            "server": "Sandman",

        }
    });

    

    if let Err(_disconnected) = workers
        .read().await
        .get(&id_).unwrap()
        .send(Ok(Message::text(query.to_string()))) {
            log_info(
                format!(
                    "{} {}", "[ ERROR ]".red(),
                    "Worker has lost connection to reciever, restart Sandman.".red()
                )
            );
        }

    loop { 
        if cache.read().await.contains_key(&sys_id) { break }   // exit the wait for
        delay_for(Duration::from_micros(5)).await;  // This lets us hand back some control to the event loop
    }

    let data = cache.read().await;    
    match data.get(&sys_id) {
        Some(_) => {
            let asgi_resp = data.get(&sys_id).unwrap();
            let headers: Vec<Vec<String>> = asgi_resp.headers.clone();
            let status: u16 = asgi_resp.status;
      
            let mut resp = Response::builder()
                .status(StatusCode::from_u16(status).unwrap());

            for v in headers.iter() {
                resp = resp.header(v[0].as_str(), v[1].as_str()) 
            }
            resp.body(asgi_resp.clone().body)
        },
        _ => {
            let resp = Response::builder()
                .status(StatusCode::from_u16(503).unwrap())
                .body("No cache".into()).unwrap();  
            Ok(resp)
        }
    } 
}


// Logging
fn log_info(content: String) {    
    let dt: DateTime<Utc> = Utc::now();
    println!("[ {} ]{}", dt.to_rfc2822().magenta(), content);
}
