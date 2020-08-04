mod boilerplate;

fn main() {
    let target = boilerplate::Target{
        file_path: String::from("./client.py"),
        app: String::from("app"),
    };

    let server_config = boilerplate::server::Config{
        addr: "127.0.0.1",
        port: 8000
    };

    println!("[ Controller ]Starting Sandman http://{}:{}", server_config.addr, server_config.port);
    boilerplate::create_forks(2, target, server_config)
}

