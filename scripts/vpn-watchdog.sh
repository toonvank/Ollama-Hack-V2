#!/bin/bash
# Keep Gluetun + backend in sync after restarts (network_mode: service:gluetun).
# Install via scripts/install-ollama-hack-service.sh (systemd timer, every minute).
set -euo pipefail

GLUETUN=ollama-hack-gluetun
BACKEND=ollama-hack-v2-backend-1
HEALTH_URL=http://127.0.0.1:8000/api/v2/health
COMPOSE=/root/Ollama-Hack-V2/scripts/compose-prod.sh

log() { echo "[vpn-watchdog $(date -Iseconds)] $*"; }

container_exists() {
  docker inspect "$1" >/dev/null 2>&1
}

health_status() {
  docker inspect "$1" --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' 2>/dev/null || echo missing
}

started_at() {
  docker inspect "$1" --format '{{.State.StartedAt}}' 2>/dev/null || echo ""
}

if ! container_exists "$GLUETUN"; then
  log "gluetun missing — bringing stack up"
  "$COMPOSE" up -d gluetun
  exit 0
fi

gluetun_state=$(docker inspect "$GLUETUN" --format '{{.State.Status}}' 2>/dev/null || echo missing)
if [[ "$gluetun_state" != "running" ]]; then
  log "gluetun not running (state=$gluetun_state) — starting"
  docker start "$GLUETUN" || "$COMPOSE" up -d gluetun
  exit 0
fi

gh=$(health_status "$GLUETUN")
if [[ "$gh" == "unhealthy" ]]; then
  log "gluetun unhealthy — restarting"
  docker restart "$GLUETUN"
  exit 0
fi

if [[ "$gh" != "healthy" ]]; then
  # Still starting (health: starting/none) — wait for next run
  exit 0
fi

if ! container_exists "$BACKEND"; then
  log "backend missing — starting stack"
  "$COMPOSE" up -d backend
  exit 0
fi

backend_state=$(docker inspect "$BACKEND" --format '{{.State.Status}}' 2>/dev/null || echo missing)
if [[ "$backend_state" != "running" ]]; then
  log "backend not running — starting"
  "$COMPOSE" up -d backend
  exit 0
fi

# Backend needs time for DB migrations on cold start — avoid restart loops.
b_uptime_sec=$(docker inspect "$BACKEND" --format '{{.State.StartedAt}}' 2>/dev/null | xargs -I{} date -d {} +%s 2>/dev/null || echo 0)
now_sec=$(date +%s)
if [[ "$b_uptime_sec" -gt 0 && $((now_sec - b_uptime_sec)) -lt 120 ]]; then
  exit 0
fi

g_start=$(started_at "$GLUETUN")
b_start=$(started_at "$BACKEND")
if [[ -n "$g_start" && -n "$b_start" && "$g_start" > "$b_start" ]]; then
  log "gluetun restarted after backend — restarting backend"
  docker restart "$BACKEND"
  exit 0
fi

if ! curl -sf --max-time 8 "$HEALTH_URL" >/dev/null; then
  log "backend health check failed — restarting backend"
  docker restart "$BACKEND"
  exit 0
fi