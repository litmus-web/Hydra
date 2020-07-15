use tokio::net::{TcpListener, TcpStream};
use tokio::io::{AsyncWriteExt, AsyncReadExt};
use std::error::Error;
use std::sync::{
    Arc
};
use futures::{StreamExt};
use warp::{Filter};
use tokio::sync::RwLock;

type Workers = Arc<RwLock<Vec<TcpStream>>>;
type Count = Arc<RwLock<usize>>;


#[tokio::main]
async fn main() -> Result<(), Box<dyn Error>> {
    let addr = "127.0.0.1:9090";
    let mut listener = TcpListener::bind(addr).await.unwrap();

    let workers_lock = Workers::default();
    let workers_clone = workers_lock.clone();

    let counter_lock = Count::default();
    let counter_clone = counter_lock.clone();

    let counter_clone = warp::any().map(move || counter_clone.clone());
    let workers_clone = warp::any().map(move || workers_clone.clone());

    let routes = warp::path("bob").and(workers_clone.clone()).and(counter_clone.clone()).and_then(test);

    let server = async move {
        let mut incoming = listener.incoming();
        while let Some(socket_res) = incoming.next().await {
            match socket_res {
                Err(err) => {
                    println!("Error handling socket {:?}", err);
                }
                Ok(mut socket) => {
                    println!("🦄  Accepted connection from addr: {:?}", socket.peer_addr());
                    let _ = socket.write_all(b"hello world!").await;
                    workers_lock.write().await.push(socket);
                }                
            }
        }
    };
    println!("Sandman server starting up... Waiting for workers to spin up.");

    // Start the server and block the fn until `server` spins down.
    tokio::spawn( async move { server.await; });

    println!("====================================================");
    println!("❤️  Sandman server running on http://127.0.0.1:8080.");
    println!("====================================================");

    
    warp::serve(routes).run(([0, 0, 0, 0], 8080)).await;
    Ok(())
}

async fn test(workers: Workers, counter: Count) -> Result<impl warp::Reply, warp::Rejection> {
    let content  = get_resp(workers, counter, b"hello world--eof--").await;
    match content {
        Ok(val) => Ok(val),
        _ => Ok("Oops! The server has got itself into trouble.".to_owned())
    }    
}

async fn get_resp(workers: Workers, counter: Count, content: &'static [u8]) -> Result<String, Box<dyn Error>> {
    let mut worker_vec = workers.write().await;   
    let mut count = counter.write().await;     
    if *count < (worker_vec.len() - 1) {
        *count += 1;
    } else {
        *count = 0;
    }
    let _ = worker_vec[*count].write_all(content).await;
    let mut resp = String::new();

    loop {
        let mut buf = [0; 1024];
        let n = worker_vec[*count].read(&mut buf).await?;
        if n == 0 { break }
        let temp = String::from_utf8(buf[0..n].to_vec())?;
        resp = format!("{}{}", resp, temp);
        if resp.contains("--eom--") {
            break
         }
    }
    Ok(resp)
}