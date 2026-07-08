use std::collections::HashMap;
use std::time::{Duration, Instant};

use anyhow::{Context, Result}; // Context used by next_chunk
use bytes::Bytes;
use reqwest::header::{HeaderMap, HeaderName, HeaderValue};
use reqwest::{Client, Method, Response};
use serde::{Deserialize, Serialize};

pub mod probe;
pub mod race;
pub use probe::{
    ProbeBatchRequest, ProbeBatchResponse, ProbeRequest, ProbeResult,
};
pub use race::{join_endpoint_path, prefixed_byte_stream, RaceFailureMeta, RaceRequest, RaceWinner};

mod base64_bytes {
    use base64::{engine::general_purpose::STANDARD, Engine as _};
    use serde::{Deserialize, Deserializer, Serializer};

    pub fn serialize<S>(bytes: &Vec<u8>, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serializer.serialize_str(&STANDARD.encode(bytes))
    }

    pub fn deserialize<'de, D>(deserializer: D) -> Result<Vec<u8>, D::Error>
    where
        D: Deserializer<'de>,
    {
        let s = String::deserialize(deserializer)?;
        STANDARD
            .decode(s)
            .map_err(serde::de::Error::custom)
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RelayTimeouts {
    #[serde(default = "default_dial_ms")]
    pub dial_ms: u64,
    #[serde(default = "default_first_byte_ms")]
    pub first_byte_ms: u64,
    #[serde(default = "default_total_ms")]
    pub total_ms: u64,
}

fn default_dial_ms() -> u64 {
    5_000
}
fn default_first_byte_ms() -> u64 {
    30_000
}
fn default_total_ms() -> u64 {
    600_000
}

impl Default for RelayTimeouts {
    fn default() -> Self {
        Self {
            dial_ms: default_dial_ms(),
            first_byte_ms: default_first_byte_ms(),
            total_ms: default_total_ms(),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RelayRequest {
    pub method: String,
    pub upstream_url: String,
    #[serde(default)]
    pub headers: HashMap<String, String>,
    #[serde(default, with = "base64_bytes")]
    pub body: Vec<u8>,
    #[serde(default)]
    pub timeouts: RelayTimeouts,
}

#[derive(Debug)]
pub struct RelayStarted {
    pub status: u16,
    pub headers: HeaderMap,
    pub ttfb_ms: u64,
    pub upstream: Response,
}

#[derive(Clone)]
pub struct RacerClient {
    pub(crate) http: Client,
}

impl RacerClient {
    pub fn new() -> Result<Self> {
        let http = Client::builder()
            .tcp_keepalive(Duration::from_secs(30))
            .pool_max_idle_per_host(32)
            .connect_timeout(Duration::from_secs(10))
            .redirect(reqwest::redirect::Policy::none())
            .build()
            .context("build reqwest client")?;
        Ok(Self { http })
    }

    pub async fn start_relay(&self, req: RelayRequest) -> Result<RelayStarted> {
        let method: Method = req
            .method
            .parse()
            .with_context(|| format!("invalid HTTP method {}", req.method))?;

        let mut headers = HeaderMap::new();
        for (key, value) in req.headers {
            if key.eq_ignore_ascii_case("host") || key.eq_ignore_ascii_case("content-length") {
                continue;
            }
            let name = HeaderName::from_bytes(key.as_bytes())
                .with_context(|| format!("invalid header name {key}"))?;
            let val = HeaderValue::from_str(&value)
                .with_context(|| format!("invalid header value for {key}"))?;
            headers.insert(name, val);
        }

        let total_timeout = Duration::from_millis(req.timeouts.total_ms);
        let started = Instant::now();

        let upstream = self
            .http
            .request(method, &req.upstream_url)
            .headers(headers)
            .body(req.body)
            .timeout(total_timeout)
            .send()
            .await
            .with_context(|| format!("upstream request to {}", req.upstream_url))?;

        let status = upstream.status().as_u16();
        let upstream_headers = upstream.headers().clone();
        let ttfb_ms = started.elapsed().as_millis() as u64;

        Ok(RelayStarted {
            status,
            headers: upstream_headers,
            ttfb_ms,
            upstream,
        })
    }
}

pub async fn next_chunk(upstream: &mut Response) -> Result<Option<Bytes>> {
    match upstream.chunk().await.context("read upstream chunk")? {
        Some(chunk) => Ok(Some(chunk)),
        None => Ok(None),
    }
}