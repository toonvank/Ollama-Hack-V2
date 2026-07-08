use std::sync::Arc;

use axum::body::Body;
use axum::extract::State;
use axum::http::{HeaderMap, HeaderName, HeaderValue, StatusCode};
use axum::response::{IntoResponse, Response};
use axum::routing::{get, post};
use axum::{Json, Router};
use base64::{engine::general_purpose::STANDARD, Engine as _};
use racer_core::{
    prefixed_byte_stream, RaceRequest, RacerClient, RelayRequest,
};
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
    version: &'static str,
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
        .route("/race", post(race))
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
        version: "phase2",
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
    let status = StatusCode::from_u16(started.status).unwrap_or(StatusCode::BAD_GATEWAY);

    let mut headers = copy_upstream_headers(&started.headers);
    headers.insert(
        HeaderName::from_static("x-relay-ttfb-ms"),
        HeaderValue::from_str(&started.ttfb_ms.to_string())?,
    );
    headers.insert(
        HeaderName::from_static("x-relay-upstream"),
        HeaderValue::from_str(&upstream_url)?,
    );

    let stream = racer_core::prefixed_byte_stream(started.upstream, bytes::Bytes::new());
    Ok((status, headers, Body::from_stream(stream)).into_response())
}

async fn race(State(state): State<AppState>, Json(req): Json<RaceRequest>) -> Response {
    match state.client.race_endpoints(req).await {
        Ok(winner) => race_winner_response(winner).unwrap_or_else(|err| {
            tracing::error!("race response build failed: {err:#}");
            (
                StatusCode::BAD_GATEWAY,
                Json(serde_json::json!({ "error": err.to_string() })),
            )
                .into_response()
        }),
        Err(err) => {
            tracing::warn!("race failed: {err:#}");
            (
                StatusCode::BAD_GATEWAY,
                Json(serde_json::json!({ "error": err.to_string() })),
            )
                .into_response()
        }
    }
}

fn race_winner_response(winner: racer_core::RaceWinner) -> anyhow::Result<Response> {
    let status = StatusCode::from_u16(winner.status).unwrap_or(StatusCode::BAD_GATEWAY);
    let mut headers = copy_upstream_headers(&winner.headers);

    headers.insert(
        HeaderName::from_static("x-race-winner"),
        HeaderValue::from_str(&winner.winner)?,
    );
    headers.insert(
        HeaderName::from_static("x-race-ttfb-ms"),
        HeaderValue::from_str(&winner.ttfb_ms.to_string())?,
    );
    headers.insert(
        HeaderName::from_static("x-race-losers-cancelled"),
        HeaderValue::from_str(&winner.losers_cancelled.to_string())?,
    );

    if !winner.failures.is_empty() {
        let failures_json = serde_json::to_string(&winner.failures)?;
        let encoded = STANDARD.encode(failures_json.as_bytes());
        headers.insert(
            HeaderName::from_static("x-race-failures-b64"),
            HeaderValue::from_str(&encoded)?,
        );
    }

    let stream = prefixed_byte_stream(winner.upstream, winner.prefix);
    Ok((status, headers, Body::from_stream(stream)).into_response())
}

fn copy_upstream_headers(upstream: &HeaderMap) -> HeaderMap {
    let mut headers = HeaderMap::new();
    for (name, value) in upstream.iter() {
        if let Ok(v) = HeaderValue::from_str(value.to_str().unwrap_or("")) {
            headers.insert(name.clone(), v);
        }
    }
    headers
}