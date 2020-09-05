extern crate clap;
use clap::{Arg, App, ArgMatches};

use std::thread;

mod web_server;

/// The start of the application, this first gets any command line flags and parses them
/// using `get_flags()`.<br><br>
///
/// The amount of workers are determined by the flags or default to 1
/// in the case of windows the server can *only* bind one worker to a port, due to a limitation
/// of rust's networking IO it is hard coded to only bind with SO_REUSEADDR.<br><br>
///
/// Each 'worker' on  Rust's side is a new thread and held internally instead of python's separate
/// processes, because we dont have the issue of the GIL.
///
fn main() {
    let matches = get_flags();

    let adapter: String = matches.value_of("adapter").unwrap().parse().unwrap();

    let mut worker_count: usize = 1;
    if cfg!(unix) {
        worker_count = matches.value_of("workers").unwrap().parse().unwrap();
    }

    let server_config = web_server::server::Config{
        addr: matches.value_of("host").unwrap().parse().unwrap(),
        port: matches.value_of("port").unwrap().parse().unwrap(),
    };


    for i in 1..worker_count {
        let temp_clone = server_config.clone();
        thread::spawn(move || {
            spawn_processes(i, 11234, temp_clone.clone())
        });
    }

    spawn_processes(0, 11234, server_config.clone())
}

/// Uses clap to parse any CLI flags and returns them.
fn get_flags() -> ArgMatches<'static> {
     App::new("Sandman universal HTTP server.")
         .version("0.0.1")
         .author("Harrison Burt")
         .about("The HTTP server for all your needs.")

         .arg(Arg::with_name("host")
             .short("h")
             .long("host")
             .value_name("HOST_ADDRESS")
             .help("Set the host address e.g 127.0.0.1")
             .default_value("127.0.0.1")
             .takes_value(true))

         .arg(Arg::with_name("port")
             .short("p")
             .long("port")
             .value_name("PORT")
             .help("Set the server port")
             .default_value("8080")
             .takes_value(true))

         .arg(Arg::with_name("adapter")
             .long("adapter")
             .value_name("ADAPTER")
             .help("The adapter for WSGI, ASGI or RAW interfaces.")
             .default_value("raw")
             .takes_value(true))

         .arg(Arg::with_name("workers")
             .short("w")
             .long("workers")
             .value_name("WORKER_COUNT")
             .help("The amount of worker processes to bind (POSIX ONLY)")
             .default_value("1")
             .takes_value(true))

         .get_matches()
}

/// spawn_processes is responsible for starting the boilerplate setup that intern runs the server
///
/// **Example**
/// ```
/// let thread_no: usize = 0;
/// let open_port: usize = 1234;
/// let server_config = web_server::server::Config{
///     addr: String::from("127.0.0.1"),
///     port: 8080
/// }
///
/// spawn_processes(thread_no, open_port, server_config);
///
/// ```
fn spawn_processes(thead_no: usize, open_port: usize, server_config: web_server::server::Config) {
    println!(
        "[ Thread {} ] Starting Sandman worker binding to ws://127.0.0.1:{}/workers",
        thead_no,
        open_port
    );
    web_server::create_boilerplate(open_port, server_config)
}

