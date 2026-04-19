use axum::{
    extract::{
        ws::{Message, WebSocket, WebSocketUpgrade},
        Query, State,
    },
    response::IntoResponse,
};
use serde::Deserialize;
use serde_json::json;

use crate::state::AppState;
use crate::features::auth::auth_util::jwt;

#[derive(Deserialize)]
pub struct WsParams {
    token: Option<String>,
}

pub async fn handle(
    ws: WebSocketUpgrade,
    Query(params): Query<WsParams>,
    State(state): State<AppState>,
) -> impl IntoResponse {
    // Validate token before upgrading
    let user_id = params
        .token
        .as_deref()
        .and_then(|t| jwt::decode_access(t, &state.config.jwt.secret).ok())
        .map(|c| c.sub);

    if user_id.is_none() {
        return axum::http::StatusCode::UNAUTHORIZED.into_response();
    }

    ws.on_upgrade(move |socket| handle_socket(socket, state))
}

async fn handle_socket(mut socket: WebSocket, state: AppState) {
    // Wait for client messages to subscribe/unsubscribe to job channels
    while let Some(Ok(msg)) = socket.recv().await {
        let text = match msg {
            Message::Text(t) => t,
            Message::Close(_) => break,
            _ => continue,
        };

        let Ok(v) = serde_json::from_str::<serde_json::Value>(&text) else {
            continue;
        };

        let msg_type = v["type"].as_str().unwrap_or("");
        let job_id   = v["job_id"].as_i64();

        match (msg_type, job_id) {
            ("subscribe", Some(jid)) => {
                let mut rx = state.hub.subscribe(jid);
                let mut sock_tx = socket;

                // Forward hub messages to this WebSocket until it closes
                loop {
                    tokio::select! {
                        event = rx.recv() => {
                            match event {
                                Ok(payload) => {
                                    if sock_tx.send(Message::Text(payload.into())).await.is_err() {
                                        break;
                                    }
                                }
                                Err(_) => break,
                            }
                        }
                        incoming = sock_tx.recv() => {
                            match incoming {
                                Some(Ok(Message::Close(_))) | None => break,
                                _ => {}
                            }
                        }
                    }
                }
                return;
            }
            ("ping", _) => {
                let pong = json!({"type": "pong"}).to_string();
                if socket.send(Message::Text(pong.into())).await.is_err() {
                    break;
                }
            }
            _ => {}
        }
    }
}
