use std::str;
use std::io;

use futures::{FutureExt, StreamExt};
use futures::prelude::*;

use tokio::sync::mpsc;

use tokio_tungstenite::{tungstenite::protocol, WebSocketStream};

use hyper::header::{self, AsHeaderName, HeaderMap, HeaderValue};
use hyper::{self, Body, Request, Response, StatusCode};
use hyper::upgrade::Upgraded;

use headers::{self, HeaderMapExt};

use serde_json;

use crate::web_server::structs::*;


pub async fn handle_ws_connection(
    req: Request<Body>,
    workers: Workers,
    cache: Cache,
    ) -> Result<Response<Body>, io::Error> {
    let res = match upgrade_connection(req).await {
        Err(res) => res,
        Ok((res, ws)) => {
            let run_ws_task = async {
                match ws.await {
                    Ok(ws) => {
                        handle_ws(ws, workers, cache).await;
                    }
                    Err(_e) => eprintln!("WS error"),
                }
            };
            tokio::task::spawn_local(run_ws_task);
            res
        }
    };
    Ok(res)
}

async fn handle_ws(ws: WebSocketStream<Upgraded>, workers: Workers, cache: Cache,) {
    // Split the ws connection into sender and reciever...
    let (ws_tx, mut ws_rc) = ws.split();
    let (tx, rx) = mpsc::unbounded_channel();

    tokio::task::spawn_local(rx.forward(ws_tx).map(|result| {
        if let Err(e) = result {
            eprintln!("websocket send error: {}", e);
        }
    }));

    workers.borrow_mut().insert(String::from("main"), tx);

    // Run it until something breaks or it stops normally.
    while let Some(result) = ws_rc.next().await {
        let msg = match result {
            Ok(protocol::Message::Text(text)) => text,
            Ok(protocol::Message::Binary(text)) => String::from_utf8(text).unwrap(),
            Err(e) => {
                eprintln!("websocket error {}", e);
                break;
            },
            _ => String::from("None")
        };

        let outgoing: ASGIResponse = serde_json::from_str(msg.as_str()).unwrap();

        cache
            .borrow_mut()
            .insert(
                outgoing
                    .clone()
                    .request_id,
                outgoing
            );
    }
}

fn header_matches<S: AsHeaderName>(headers: &HeaderMap<HeaderValue>, name: S, value: &str) -> bool {
    headers
        .get(name)
        .and_then(|v| v.to_str().ok())
        .map(|v| v.to_lowercase() == value)
        .unwrap_or(false)
}

pub async fn upgrade_connection(req: Request<Body>) -> Result<(
        Response<Body>,
        impl Future<Output = Result<WebSocketStream<hyper::upgrade::Upgraded>, ()>> + Send,
    ), Response<Body>> {
    // Upgrades the Request with the 3 way handshake to a websocket.

    // Generate a new response to populate later
    let mut res = Response::new(Body::empty());
    let mut header_error = false;

    if !header_matches(req.headers(), header::UPGRADE, "websocket") {
        header_error = true;
    }

    if !header_matches(req.headers(), header::SEC_WEBSOCKET_VERSION, "13") {
        header_error = true;
    }

    if !req
        .headers()
        .typed_get::<headers::Connection>()
        .map(|h| h.contains("Upgrade"))
        .unwrap_or(false)
    {
        header_error = true;
    }

    let key = req.headers().typed_get::<headers::SecWebsocketKey>();

    if key.is_none() {
        header_error = true;
    }

    if header_error {
        *res.status_mut() = StatusCode::BAD_REQUEST;
        return Err(res);
    }

    *res.status_mut() = StatusCode::SWITCHING_PROTOCOLS;
    let h = res.headers_mut();
    h.typed_insert(headers::Upgrade::websocket());
    h.typed_insert(headers::SecWebsocketAccept::from(key.unwrap()));
    h.typed_insert(headers::Connection::upgrade());

    // If this succeeds the request has upgraded otherwise errors.
    let upgraded = req
        .into_body()
        .on_upgrade()
        .map_err(|err| eprintln!("Cannot create websocket: {} ", err))
        .and_then(|upgraded| async {
            let r = WebSocketStream::from_raw_socket(
                upgraded,
                protocol::Role::Server,
                None
            ).await;
            Ok(r)
        });

    Ok((res, upgraded))
}