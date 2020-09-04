use std::collections::HashMap;
use std::str;
use std::rc::Rc;
use std::cell::RefCell;

use tokio::sync::mpsc;

use tokio_tungstenite::tungstenite;
use tokio_tungstenite::{tungstenite::protocol};

use serde::{Serialize, Deserialize};


///
///  Custom types for workers, cache, etc...
/// 
///     - Workers: Set in a HashMap contained in a RC to allow us to use Default().
///     - Thread Safety: This is fine because the system is Single threaded.
pub type Workers = Rc<RefCell<HashMap<
    String, mpsc::UnboundedSender<Result<protocol::Message, tungstenite::error::Error>>>>>;
pub type Cache = Rc<RefCell<HashMap<usize, ASGIResponse>>>;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResponseMeta {
    pub response_type: String
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ASGIResponse {
    pub op: usize,              // Operation Code
    pub meta: ResponseMeta,     // Response Metadata needed for responses
    pub request_id: usize,      // req id needed to go to cache

    pub response_type: String,
    pub status: u16,
    pub headers: Vec<Vec<String>>,
    pub body: String,
    pub more_body: bool,
}

#[derive(Serialize, Deserialize)]
pub struct ShardIdentify {
    pub op: usize,
    pub shard_id: String
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OutGoingRequest {
    pub op: usize,              // Operation Code

    pub request_id: usize,
    pub method: String,
    pub remote: String,
    pub path: String,
    pub headers: HashMap<String, String>,
    pub version: String,
    pub body: String,
    pub query: String,
}