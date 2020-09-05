use std::collections::HashMap;
use std::str;
use std::rc::Rc;
use std::cell::RefCell;

use tokio::sync::mpsc;

use tokio_tungstenite::tungstenite;
use tokio_tungstenite::{tungstenite::protocol};

use serde::{Serialize, Deserialize};


/// A RefCell used for storing any worker senders, anything sent to the sender
/// is then forwarded directly to the websocket, each shard is added to the hashmap to
/// make for easy change of load balancing.
pub type Workers = Rc<RefCell<HashMap<
    String, mpsc::UnboundedSender<Result<protocol::Message, tungstenite::error::Error>>>>>;

/// Used for storing any responses from workers, any pending requests are waiting for these before
/// sending the response back.
pub type Cache = Rc<RefCell<HashMap<usize, ASGIResponse>>>;

/// Represents any response metadata, this is because the system supports
/// complete or partial responses that will have a impact on performance.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResponseMeta {
    pub response_type: String
}

/// The main response coming back from a worker.
///
/// - op - The operation code: `usize`
/// - meta - A ResponseMeta struct: `ResponseMeta`
/// - request_id - A usize id originally made by the server: `usize`
///
/// - response_type - Similar to ASGI response type: `String`
/// - status - The HTTP status code: `u16`
/// - headers - A vector or vectors (depth 2): `Vec<Vec<String>>`
/// - body - The response body, this can either be partial or full: `String`
/// - more_body - A bool to represent if there is more responses to come for this shard: `bool`
///
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

/// Used for a shard identify as the first incoming response.
///
/// - op - The operation code: `usize`
/// - shard_id - The shard id of the connection: `String`
///
#[derive(Serialize, Deserialize)]
pub struct ShardIdentify {
    pub op: usize,
    pub shard_id: String
}

/// This represents the request that gets sent to the WS by the server itself
///
/// - op - The operation code: `usize`
///
/// - request_id - A usize id originally made by the server: `usize`
/// - method - The HTTP method: `String`
/// - remote - The remote HTTP addr: `String`
/// - path - The URL path of the HTTP request `String`
/// - headers - The HTTP headers: `HashMap<String, String>`
/// - version - The HTTP protocol verison e.g HTTP/1.1: `String`
/// - body - The request body: `String`
/// - query - The URL query e.g ?abc=dfg: `String`
///
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