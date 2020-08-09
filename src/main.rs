use std::env;
use std::process;

mod boilerplate;

fn main() {
    let pid = process::id();
    let args: Vec<String> = env::args().collect();

    let proc_id: usize = args[1].clone().parse().unwrap_or(1234);
    let addr: String = args[2].clone();
    let port: u16 = args[3].clone().parse().unwrap_or(8000);

    let server_config = boilerplate::server::Config{
        addr,
        port
    };

    println!("[{}] Starting Sandman worker binding to ws://127.0.0.1:{}/workers", pid, proc_id);
    boilerplate::create_boilerplate(proc_id, server_config)
}

