use std::thread;

pub mod server;


pub fn create_boilerplate(process_id: usize, server_config: server::Config) {
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
        server::run(
            process_id,
            server_config,        
        )
    );
}

