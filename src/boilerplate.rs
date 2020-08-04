use std::thread;

pub mod server;


#[derive(Clone)]
pub struct Target {
    pub file_path: String,
    pub app: String,
}

pub fn create_forks(amount_of_processes: usize, target: Target, server_config: server::Config) {
    let workers = server::Workers::default();

    for i in 1..amount_of_processes {
        let t = target.clone();
        let c = server_config.clone();
        let w = workers.clone();
        let _ = thread::Builder::new().name(format!("fork_{}", i).to_string()).spawn(move || {
            create_boilerplate(i, t, c, w)
        });
    }
    create_boilerplate(0, target.clone(), server_config.clone(), workers.clone())
}

fn create_boilerplate(process_id: usize, target: Target, server_config: server::Config, workers: server::Workers) {
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
            workers
        )
    );
}

