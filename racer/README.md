# ollama-racer

Rust sidecar for Ollama-Hack egress hot path (stream relay → parallel race → probes).

## Phase 1 — stream relay

- `GET /health` — liveness
- `POST /relay` — single upstream HTTP relay with streaming response

Go calls relay when `RACER_RELAY_ENABLED=true` (default **off**).

## Phase 2 — parallel race

- `POST /race` — fan out to N endpoints, cancel losers on first valid TTFB, stream winner

Go calls race when `RACER_RACE_ENABLED=true` (default **off**). Takes priority over relay.

## Phase 3 — background probes

- `POST /probe` — single endpoint health check (`GET /api/version` by default)
- `POST /probe/batch` — bounded-concurrency batch probes (`RACER_PROBE_CONCURRENCY`, default 50)

Go routes health-tracker probes and tester outbound I/O when `BACKGROUND_ENDPOINT_OUTBOUND=rust`.
Tester streaming tests use `POST /relay` under the hood. Go still owns Postgres scores.

### Local build

```bash
cd racer && cargo build --release
RACER_BIND=127.0.0.1:8787 ./target/release/ollama-racer
```

### Docker (prod + VPN)

```bash
RACER_COMPOSE=1 bash scripts/compose-prod.sh up -d --build racer
curl -s http://127.0.0.1:8787/health   # from Gluetun netns / published port
```

### Enable in Go

```bash
# Phase 3 (background probes + tester via VPN netns)
BACKGROUND_ENDPOINT_OUTBOUND=rust
# Phase 2 (parallel race — recommended once smoke-tested)
RACER_RACE_ENABLED=true
# Phase 1 (single-endpoint relay only)
RACER_RELAY_ENABLED=true
RACER_URL=http://127.0.0.1:8787   # same Gluetun netns on prod
```

Response headers: `X-Race-Winner`, `X-Race-Ttfb-Ms`, `X-Race-Losers-Cancelled`, `X-Race-Failures-B64`.

Rollback: set flags to `false` — instant fallback to Go race path.