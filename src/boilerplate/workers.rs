use tokio::process::{Command};

pub struct WorkerConfig {
    command: &'static str
}

pub struct WorkerSettings {
    id: &'static str,
    port: &'static str,
    adapter: &'static str,
}

async fn spawn_worker(config: WorkerConfig, settings: WorkerSettings) {
    let child = Command::new(config.command).arg("--port").arg(settings.port)
                        .spawn();

}