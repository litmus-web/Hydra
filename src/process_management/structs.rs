#[derive(Debug, Clone)]
pub struct WorkerConfig {
    pub process_settings: ProcessSettings,
    pub worker_settings: WorkerSettings,
}

#[derive(Debug, Clone)]
pub struct ProcessSettings {
    pub command: &'static str,
}

#[derive(Debug, Clone)]
pub struct WorkerSettings {
    pub id: &'static str,
    pub port: &'static str,
    pub adapter: &'static str,
}
