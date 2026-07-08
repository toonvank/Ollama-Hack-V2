# ollama-racer

Rust sidecar for Ollama-Hack egress hot path (stream relay → parallel race → probes).

## Phase 1 — stream relay

- `GET /health` — liveness
- `POST /relay` — single upstream HTTP relay with streaming response

Go control plane calls the sidecar when `RACER_RELAY_ENABLED=true` (default **off**).

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

### Enable relay in Go

```bash
RACER_RELAY_ENABLED=true
RACER_URL=http://127.0.0.1:8787   # same Gluetun netns on prod
```

Rollback: set `RACER_RELAY_ENABLED=false` — instant fallback to Go race path.