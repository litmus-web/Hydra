use tokio::process::{Command};

use crate::process_management::structs;

pub async fn start_worker(config: structs::WorkerConfig) {
    spawn_worker(config.worker_settings, config.process_settings);

}

async fn spawn_worker(config: structs::WorkerSettings, settings: structs::ProcessSettings) {
    println!("hello world");

    let child = Command::new(settings.command).arg("--port").arg(config.port)
                        .spawn();

}