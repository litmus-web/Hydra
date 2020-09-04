pub mod structs;
pub mod websocket;
pub mod server;
pub mod workers;


pub fn create_boilerplate(free_port: usize, server_config: server::Config) {
     // Configure a runtime that runs everything on the current thread
     let mut rt = tokio::runtime::Builder::new()
     .enable_all()
     .basic_scheduler()
     .build()
     .expect("build runtime");

    // Combine it with a `LocalSet,  which means it can spawn !Send futures...
    let local = tokio::task::LocalSet::new();
    let _ = local.block_on(
        &mut rt,
        async_run(
            free_port,
            server_config,
        )
    );
}

async fn async_run(free_port: usize, server_config: server::Config) {
    server::run(free_port, server_config).await;
}

