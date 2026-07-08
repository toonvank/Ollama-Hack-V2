use std::collections::HashMap;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};

use anyhow::{anyhow, Context, Result};
use bytes::Bytes;
use reqwest::header::{HeaderMap, HeaderName, HeaderValue};
use reqwest::{Client, Method, Response};
use serde::{Deserialize, Serialize};
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;

use crate::{RelayTimeouts, RacerClient};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RaceRequest {
    pub method: String,
    pub path: String,
    pub endpoints: Vec<String>,
    #[serde(default)]
    pub headers: HashMap<String, String>,
    #[serde(default, with = "crate::base64_bytes")]
    pub body: Vec<u8>,
    #[serde(default)]
    pub timeouts: RelayTimeouts,
    #[serde(default)]
    pub stream: bool,
    #[serde(default = "default_true")]
    pub cancel_on_first_win: bool,
}

fn default_true() -> bool {
    true
}

#[derive(Debug, Clone, Serialize)]
pub struct RaceFailureMeta {
    pub endpoint: String,
    pub status: u16,
    pub quota_exceeded: bool,
    pub rate_limited: bool,
    pub client_error: bool,
    pub message: String,
}

#[derive(Debug)]
pub struct RaceWinner {
    pub winner: String,
    pub status: u16,
    pub headers: HeaderMap,
    pub ttfb_ms: u64,
    pub losers_cancelled: usize,
    pub failures: Vec<RaceFailureMeta>,
    pub upstream: Response,
    pub prefix: Bytes,
}

pub fn join_endpoint_path(base: &str, path: &str) -> String {
    let base = base.trim_end_matches('/');
    if path.is_empty() {
        return base.to_string();
    }
    if path.starts_with('/') {
        return format!("{base}{path}");
    }
    format!("{base}/{path}")
}

fn max_fanout() -> usize {
    std::env::var("RACER_MAX_FANOUT")
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(32)
}

fn is_quota_exceeded(status: u16, body: &[u8]) -> bool {
    let body_lc = String::from_utf8_lossy(body).to_ascii_lowercase();
    const KEYWORDS: &[&str] = &[
        "insufficient_quota",
        "insufficient_balance",
        "insufficient balance",
        "quota_exceeded",
        "quota exceeded",
        "exceeded your current quota",
        "billing_not_active",
        "out of balance",
        "run out of balance",
        "credit limit reached",
        "free_trial_quota",
        "requires a subscription",
        "requires subscription",
        "subscription required",
        "upgrade for access",
        "ollama.com/upgrade",
        "session usage limit",
    ];
    KEYWORDS.iter().any(|kw| body_lc.contains(kw)) || (status == 402)
}

struct CandidateSuccess {
    endpoint: String,
    status: u16,
    headers: HeaderMap,
    upstream: Response,
    prefix: Bytes,
    ttfb_ms: u64,
}

enum CandidateOutcome {
    Win(CandidateSuccess),
    Fail(RaceFailureMeta),
}

impl RacerClient {
    pub async fn race_endpoints(&self, req: RaceRequest) -> Result<RaceWinner> {
        if req.endpoints.is_empty() {
            return Err(anyhow!("no endpoints to race"));
        }

        let fanout = req.endpoints.len().min(max_fanout());
        let endpoints: Vec<String> = req.endpoints.iter().take(fanout).cloned().collect();
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

        let cancel = CancellationToken::new();
        let (tx, mut rx) = mpsc::channel::<CandidateOutcome>(fanout * 2);
        let http: Client = self.http.clone();
        let body = req.body.clone();
        let path = req.path.clone();
        let stream = req.stream;
        let total_timeout = Duration::from_millis(req.timeouts.total_ms);
        let started = Arc::new(AtomicUsize::new(0));

        for endpoint in endpoints {
            let tx = tx.clone();
            let cancel = cancel.clone();
            let http = http.clone();
            let headers = headers.clone();
            let body = body.clone();
            let path = path.clone();
            let method = method.clone();
            started.fetch_add(1, Ordering::SeqCst);

            tokio::spawn(async move {
                if cancel.is_cancelled() {
                    let _ = tx.try_send(CandidateOutcome::Fail(RaceFailureMeta {
                        endpoint,
                        status: 0,
                        quota_exceeded: false,
                        rate_limited: false,
                        client_error: false,
                        message: "cancelled".into(),
                    }));
                    return;
                }

                let outcome = race_one(
                    &http,
                    method,
                    &endpoint,
                    &path,
                    headers,
                    body,
                    stream,
                    total_timeout,
                )
                .await;
                let _ = tx.try_send(outcome);
            });
        }
        drop(tx);

        let mut failures = Vec::new();
        let mut winner: Option<CandidateSuccess> = None;
        let total = started.load(Ordering::SeqCst);

        while let Some(outcome) = rx.recv().await {
            match outcome {
                CandidateOutcome::Win(success) => {
                    if req.cancel_on_first_win {
                        cancel.cancel();
                    }
                    winner = Some(success);
                    break;
                }
                CandidateOutcome::Fail(f) => failures.push(f),
            }
        }

        let winner = winner.ok_or_else(|| {
            anyhow!(
                "all endpoints failed the race ({} failures)",
                failures.len().max(total)
            )
        })?;

        let losers_cancelled = if req.cancel_on_first_win {
            total.saturating_sub(1)
        } else {
            total.saturating_sub(1 + failures.len())
        };

        Ok(RaceWinner {
            winner: winner.endpoint,
            status: winner.status,
            headers: winner.headers,
            ttfb_ms: winner.ttfb_ms,
            losers_cancelled,
            failures,
            upstream: winner.upstream,
            prefix: winner.prefix,
        })
    }
}

async fn race_one(
    http: &Client,
    method: Method,
    endpoint: &str,
    path: &str,
    headers: HeaderMap,
    body: Vec<u8>,
    stream_requested: bool,
    total_timeout: Duration,
) -> CandidateOutcome {
    let target = join_endpoint_path(endpoint, path);
    let started = Instant::now();

    let resp = match http
        .request(method, &target)
        .headers(headers)
        .body(body)
        .timeout(total_timeout)
        .send()
        .await
    {
        Ok(r) => r,
        Err(err) => {
            return CandidateOutcome::Fail(RaceFailureMeta {
                endpoint: endpoint.to_string(),
                status: 0,
                quota_exceeded: false,
                rate_limited: false,
                client_error: false,
                message: err.to_string(),
            });
        }
    };

    let status = resp.status().as_u16();
    if status >= 400 {
        let body_bytes = resp.bytes().await.unwrap_or_default();
        let quota = is_quota_exceeded(status, &body_bytes);
        return CandidateOutcome::Fail(RaceFailureMeta {
            endpoint: endpoint.to_string(),
            status,
            quota_exceeded: quota,
            rate_limited: status == 429,
            client_error: (400..500).contains(&status) && status != 429,
            message: format!("status {status}"),
        });
    }

    let content_type = resp
        .headers()
        .get(reqwest::header::CONTENT_TYPE)
        .and_then(|v| v.to_str().ok())
        .unwrap_or("")
        .to_ascii_lowercase();

    if content_type.contains("text/html") {
        resp.bytes().await.ok();
        return CandidateOutcome::Fail(RaceFailureMeta {
            endpoint: endpoint.to_string(),
            status,
            quota_exceeded: false,
            rate_limited: false,
            client_error: false,
            message: "honeypot html".into(),
        });
    }

    if stream_requested
        && !content_type.contains("event-stream")
        && !content_type.contains("ndjson")
    {
        resp.bytes().await.ok();
        return CandidateOutcome::Fail(RaceFailureMeta {
            endpoint: endpoint.to_string(),
            status,
            quota_exceeded: false,
            rate_limited: false,
            client_error: false,
            message: format!("expected stream, got {content_type}"),
        });
    }

    let mut upstream = resp;
    let prefix = match upstream.chunk().await {
        Ok(Some(chunk)) => chunk,
        Ok(None) => {
            return CandidateOutcome::Fail(RaceFailureMeta {
                endpoint: endpoint.to_string(),
                status,
                quota_exceeded: false,
                rate_limited: false,
                client_error: false,
                message: "empty body".into(),
            });
        }
        Err(err) => {
            return CandidateOutcome::Fail(RaceFailureMeta {
                endpoint: endpoint.to_string(),
                status,
                quota_exceeded: false,
                rate_limited: false,
                client_error: false,
                message: format!("read error: {err}"),
            });
        }
    };

    let n = prefix.len().min(512);
    let sniff = String::from_utf8_lossy(&prefix[..n]);
    let sniff_trim = sniff.trim();
    if !sniff_trim.is_empty() {
        let first = sniff_trim.as_bytes()[0];
        if first != b'{' && first != b'[' && first != b'd' && first != b'"' {
            return CandidateOutcome::Fail(RaceFailureMeta {
                endpoint: endpoint.to_string(),
                status,
                quota_exceeded: false,
                rate_limited: false,
                client_error: false,
                message: format!("invalid payload start {first}"),
            });
        }
        if sniff_trim.starts_with("{\"error\"")
            || sniff_trim.starts_with("{\"message\"")
        {
            let quota = is_quota_exceeded(200, sniff_trim.as_bytes());
            return CandidateOutcome::Fail(RaceFailureMeta {
                endpoint: endpoint.to_string(),
                status,
                quota_exceeded: quota,
                rate_limited: false,
                client_error: false,
                message: "200 error json".into(),
            });
        }
        if stream_requested {
            if !sniff_trim.starts_with("data:") {
                return CandidateOutcome::Fail(RaceFailureMeta {
                    endpoint: endpoint.to_string(),
                    status,
                    quota_exceeded: false,
                    rate_limited: false,
                    client_error: false,
                    message: "non-sse stream".into(),
                });
            }
            if sniff_trim.contains("\"error\"") {
                let quota = is_quota_exceeded(200, sniff_trim.as_bytes());
                return CandidateOutcome::Fail(RaceFailureMeta {
                    endpoint: endpoint.to_string(),
                    status,
                    quota_exceeded: quota,
                    rate_limited: false,
                    client_error: false,
                    message: "sse error payload".into(),
                });
            }
        }
    }

    let ttfb_ms = started.elapsed().as_millis() as u64;
    CandidateOutcome::Win(CandidateSuccess {
        endpoint: endpoint.to_string(),
        status,
        headers: upstream.headers().clone(),
        upstream,
        prefix,
        ttfb_ms,
    })
}

pub fn prefixed_byte_stream(
    mut upstream: Response,
    prefix: Bytes,
) -> impl futures_util::Stream<Item = std::result::Result<Bytes, std::io::Error>> + Send {
    async_stream::stream! {
        if !prefix.is_empty() {
            yield Ok(prefix);
        }
        loop {
            match upstream.chunk().await {
                Ok(Some(chunk)) => yield Ok(chunk),
                Ok(None) => break,
                Err(err) => {
                    yield Err(std::io::Error::other(err.to_string()));
                    break;
                }
            }
        }
    }
}