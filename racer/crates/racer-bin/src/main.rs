use std::sync::Arc;

use axum::body::Body;
use axum::extract::State;
use axum::http::{HeaderMap, HeaderName, HeaderValue, StatusCode};
use axum::response::{IntoResponse, Response};
use axum::routing::{get, post};
use axum::{Json, Router};
use bytes::Bytes;
use futures_util::StreamExt;
use racer_core::{next_chunk, RacerClient, RelayRequest};
use serde::Serialize;
use tokio::net::TcpListener;
use tower_http::trace::TraceLayer;
use tracing::info;

#[derive(Clone)]
struct AppState {
    client: Arc<RacerClient>,
}

#[derive(Serialize)]
struct HealthResponse {
    status: &'static str,
    service: &'static str,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "info,racer_bin=info".into()),
        )
        .init();

    let bind = std::env::var("RACER_BIND").unwrap_or_else(|_| "0.0.0.0:8787".to_string());
    let client = Arc::new(RacerClient::new()?);
    let state = AppState { client };

    let app = Router::new()
        .route("/health", get(health))
        .route("/relay", post(relay))
        .layer(TraceLayer::new_for_http())
        .with_state(state);

    let listener = TcpListener::bind(&bind).await?;
    info!("ollama-racer listening on {bind}");
    axum::serve(listener, app).await?;
    Ok(())
}

async fn health() -> Json<HealthResponse> {
    Json(HealthResponse {
        status: "healthy",
        service: "ollama-racer",
    })
}

async fn relay(State(state): State<AppState>, Json(req): Json<RelayRequest>) -> Response {
    match relay_inner(state, req).await {
        Ok(resp) => resp,
        Err(err) => {
            tracing::error!("relay failed: {err:#}");
            (
                StatusCode::BAD_GATEWAY,
                Json(serde_json::json!({ "error": err.to_string() })),
            )
                .into_response()
        }
    }
}

async fn relay_inner(state: AppState, req: RelayRequest) -> anyhow::Result<Response> {
    let upstream_url = req.upstream_url.clone();
    let started = state.client.start_relay(req).await?;
    let status =
        StatusCode::from_u16(started.status).unwrap_or(StatusCode::BAD_GATEWAY);

    let mut headers = HeaderMap::new();
    for (name, value) in started.headers.iter() {
        if let (Ok(n), Ok(v)) = (
            HeaderName::from_bytes(name.as_str().as_bytes()),
            HeaderValue::from_str(value.to_str().unwrap_or("")),
        ) {
            headers.insert(n, v);
        }
    }
    headers.insert(
        HeaderName::from_static("x-relay-ttfb-ms"),
        HeaderValue::from_str(&started.ttfb_ms.to_string())?,
    );
    headers.insert(
        HeaderName::from_static("x-relay-upstream"),
        HeaderValue::from_str(&upstream_url)?,
    );

    let mut upstream = started.upstream;
    let stream = async_stream::stream! {
        loop {
            match next_chunk(&mut upstream).await {
                Ok(Some(chunk)) => yield Ok::<Bytes, std::io::Error>(chunk),
                Ok(None) => break,
                Err(err) => {
                    yield Err(std::io::Error::other(err.to_string()));
                    break;
                }
            }
        }
    };

    Ok((status, headers, Body::from_stream(stream)).into_response())
}