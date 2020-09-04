extern crate clap;
use clap::{Arg, App, ArgMatches};

// use std::process;
use std::thread;

mod boilerplate;


fn main() {
    let matches = get_flags();

    let adapter: String = matches.value_of("adapter").unwrap().parse().unwrap();

    let mut worker_count: usize = 1;
    if cfg!(unix) {
        worker_count = matches.value_of("workers").unwrap().parse().unwrap();
    }

    let server_config = boilerplate::server::Config{
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

fn spawn_processes(thead_no: usize, open_port: usize, server_config: boilerplate::server::Config) {
    println!(
        "[ Thread {} ] Starting Sandman worker binding to ws://127.0.0.1:{}/workers",
        thead_no,
        open_port
    );
    boilerplate::create_boilerplate(open_port, server_config)
}

