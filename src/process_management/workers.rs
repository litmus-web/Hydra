use tokio::process::{Command};

use crate::process_management::structs;

pub async fn start_worker(config: structs::WorkerConfig) {
    tokio::task::spawn_local(
        spawn_worker(
            config.worker_settings,
            config.process_settings
        )
    );
}

async fn spawn_worker(
    worker_settings: structs::WorkerSettings,
    process_settings: structs::ProcessSettings
) {
    println!("hello world");

    let child = Command::new(process_settings.command).arg("--port").arg(worker_settings.port)
                        .spawn();

}