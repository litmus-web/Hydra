use tokio::sync::{mpsc, RwLock};
use tokio::time::{delay_for, Duration};
use tokio::runtime::{Builder};
use tokio::net::TcpListener;
// use tokio::process::Command;

use futures::{FutureExt, StreamExt};

use serde_json::json;

use warp::ws::{Message, WebSocket};
use warp::{Filter, http::HeaderMap};
use warp::hyper;

use std::error::Error;
use std::sync::Arc;
use std::collections::HashMap;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::thread;
use std::net::Ipv4Addr;

use colored::*;

use chrono::{DateTime, Utc};

use clap::{Arg, App};



static NEXT_SYS_ID: AtomicUsize = AtomicUsize::new(1);

type WorkersSend = Arc<RwLock<HashMap<usize, mpsc::UnboundedSender<Result<Message, warp::Error>>>>>;
type Cache = Arc<RwLock<HashMap<String, serde_json::Value>>>;


#[derive(Clone)]
struct WorkerConfig {
    id: usize,
    host: Ipv4Addr,
    port: u16,
    thread_count: usize,
}

fn main() {    

    // some stuff
    let threads: usize = 1;

    let host: Ipv4Addr = Ipv4Addr::new(0, 0, 0, 0);
    let port: u16 = 8080;
    
    
    println!("{}", "=============== Starting Sandman =============".cyan());
    println!("  Worker count:  {}", threads);
    println!("  Worker path  @ ws://127.0.0.1:8080/workers");
    println!("  Server running  @ http://127.0.0.1:8080/");
    println!("{}", "==============================================".cyan());
    

    for i in 1..threads {
        let config = WorkerConfig{
            id: i,
            host: host.clone(),
            port: port,
            thread_count: threads,
        };

        let _ = thread::Builder::new().name(
            format!("sandman-worker-{}", i).to_string()).spawn(
                move || { start_workers(config) }
            );
    };    

    let config = WorkerConfig{
            id: threads,
            host: host.clone(),
            port: port,
            thread_count: threads,
    };
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
    
    // Setting some things before we start
    let target_path = String::from("file:app"); 
    let temp = target_path.split(":").collect::<Vec<&str>>();
    let _file_path = temp[0];
    let _app = temp[1];

    //let child = Command::new("python")
    //    .arg(format!("{}.py", file_path))
    //    .arg(app)
    //    .spawn();

    
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
        let mut data: serde_json::Value = serde_json::from_str(msg).unwrap();
        let key: serde_json::Value = data["id"].take();
        let key = key.as_str().unwrap();
        cache.write().await.insert(key.to_owned(), data);
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
        Ok(warp::reply::html("Webserver is not ready yet!".to_owned()))
    } else {
        let content  = get_resp(path_to_serve, cfg, headers, http_method, workers, cache).await;
        match content {
            Ok(val) => Ok(warp::reply::html(val)),
            _ =>  {
                Ok(warp::reply::html("Oops! and error has ucoured.".to_owned()))
            }
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
    ) -> Result<String, Box<dyn Error>> {

    let sys_id = NEXT_SYS_ID.fetch_add(1, Ordering::Relaxed).to_string();
    let mut header_map = HashMap::new();

    for (key, value) in headers.iter() {
        header_map.insert(
            format!("{}", key.as_str()),
             format!("{}", value.to_str().unwrap_or("None"))
            );
    }

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
        .get(&cfg.id).unwrap()
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
    if data.get(&sys_id) != None {
        Ok(data.get(&sys_id).unwrap()["content"].as_str().unwrap().to_string())
    } else {
        Ok(String::from("Bad cache"))
    }

}


// Logging
fn log_info(content: String) {    
    let dt: DateTime<Utc> = Utc::now();
    println!("[ {} ]{}", dt.to_rfc2822().magenta(), content);
}
