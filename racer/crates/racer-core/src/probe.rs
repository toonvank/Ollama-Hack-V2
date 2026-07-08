use std::sync::Arc;
use std::time::{Duration, Instant};

use anyhow::{anyhow, Result};
use reqwest::Client;
use serde::{Deserialize, Serialize};
use tokio::sync::Semaphore;

use crate::race::join_endpoint_path;
use crate::{RelayTimeouts, RacerClient};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProbeRequest {
    pub endpoint: String,
    #[serde(default = "default_probe_path")]
    pub path: String,
    #[serde(default)]
    pub timeouts: RelayTimeouts,
}

fn default_probe_path() -> String {
    "/api/version".into()
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProbeBatchRequest {
    pub endpoints: Vec<String>,
    #[serde(default = "default_probe_path")]
    pub path: String,
    #[serde(default)]
    pub timeouts: RelayTimeouts,
    #[serde(default)]
    pub max_concurrency: usize,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProbeResult {
    pub endpoint: String,
    pub status: u16,
    pub latency_ms: u64,
    pub ok: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProbeBatchResponse {
    pub results: Vec<ProbeResult>,
    pub probed: usize,
    pub duration_ms: u64,
}

fn default_probe_concurrency() -> usize {
    std::env::var("RACER_PROBE_CONCURRENCY")
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(50)
}

fn max_batch_size() -> usize {
    std::env::var("RACER_PROBE_MAX_BATCH")
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(500)
}

impl RacerClient {
    pub async fn probe_endpoint(&self, req: ProbeRequest) -> Result<ProbeResult> {
        let batch = self
            .probe_batch(ProbeBatchRequest {
                endpoints: vec![req.endpoint],
                path: req.path,
                timeouts: req.timeouts,
                max_concurrency: 1,
            })
            .await?;
        batch
            .results
            .into_iter()
            .next()
            .ok_or_else(|| anyhow!("probe returned no results"))
    }

    pub async fn probe_batch(&self, req: ProbeBatchRequest) -> Result<ProbeBatchResponse> {
        if req.endpoints.is_empty() {
            return Err(anyhow!("no endpoints to probe"));
        }

        let started = Instant::now();
        let fanout = req.endpoints.len().min(max_batch_size());
        let endpoints: Vec<String> = req.endpoints.iter().take(fanout).cloned().collect();
        let concurrency = if req.max_concurrency == 0 {
            default_probe_concurrency()
        } else {
            req.max_concurrency.min(default_probe_concurrency())
        };
        let semaphore = Arc::new(Semaphore::new(concurrency));
        let http = self.http.clone();
        let path = req.path.clone();
        let timeout = Duration::from_millis(req.timeouts.total_ms.max(1));

        let mut handles = Vec::with_capacity(endpoints.len());
        for endpoint in endpoints {
            let permit = semaphore.clone().acquire_owned().await?;
            let http = http.clone();
            let path = path.clone();
            handles.push(tokio::spawn(async move {
                let result = probe_one(&http, &endpoint, &path, timeout).await;
                drop(permit);
                result
            }));
        }

        let mut results = Vec::with_capacity(handles.len());
        for handle in handles {
            match handle.await {
                Ok(result) => results.push(result),
                Err(err) => results.push(ProbeResult {
                    endpoint: String::new(),
                    status: 0,
                    latency_ms: 0,
                    ok: false,
                    error: Some(format!("task join error: {err}")),
                }),
            }
        }

        Ok(ProbeBatchResponse {
            probed: results.len(),
            duration_ms: started.elapsed().as_millis() as u64,
            results,
        })
    }
}

async fn probe_one(http: &Client, endpoint: &str, path: &str, timeout: Duration) -> ProbeResult {
    let target = join_endpoint_path(endpoint, path);
    let started = Instant::now();

    match http.get(&target).timeout(timeout).send().await {
        Ok(resp) => {
            let status = resp.status().as_u16();
            let latency_ms = started.elapsed().as_millis() as u64;
            let ok = status >= 200 && status < 300;
            ProbeResult {
                endpoint: endpoint.to_string(),
                status,
                latency_ms,
                ok,
                error: None,
            }
        }
        Err(err) => ProbeResult {
            endpoint: endpoint.to_string(),
            status: 0,
            latency_ms: started.elapsed().as_millis() as u64,
            ok: false,
            error: Some(err.to_string()),
        },
    }
}