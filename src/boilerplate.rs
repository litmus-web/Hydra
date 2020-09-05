use crate::web_server::server;
use crate::process_management;

/// The main function that sets up all of Tokio's boiler plate.
/// because we want to create multiple threads with event loops we cannot use
/// the standard method of #[tokio::main].
/// <br><br>
/// **Scheduler settings**
/// - enable_all
/// - basic_scheduler
/// - LocalSet
///
pub fn create_boilerplate(
    free_port: usize,
    server_config: server::Config,
    worker_config: process_management::structs::WorkerConfig
) {
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
            worker_config
        )
    );
}

/// Used by `create_boilerplate` to start the server by awaiting it.
pub async fn async_run(
    free_port: usize,
    server_config: server::Config,
    worker_config: process_management::structs::WorkerConfig,
) {
    process_management::workers::start_worker(worker_config).await;
    server::run(free_port, server_config).await;
}
