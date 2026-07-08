#!/bin/bash
# Keep Gluetun healthy; backend uses Gluetun HTTP proxy for user traffic only.
# Install via scripts/install-ollama-hack-service.sh (systemd timer, every minute).
set -euo pipefail

GLUETUN=ollama-hack-gluetun
BACKEND=ollama-hack-v2-backend-1
HEALTH_URL=http://127.0.0.1:8000/api/v2/health
COMPOSE=/root/Ollama-Hack-V2/scripts/compose-prod.sh
STATE_DIR=/var/lib/ollama-hack-watchdog
GLUETUN_BAD_SINCE="$STATE_DIR/gluetun-unhealthy-since"

log() { echo "[vpn-watchdog $(date -Iseconds)] $*"; }

mkdir -p "$STATE_DIR"

restart_backend() {
  log "$1"
  docker stop "$BACKEND" --time 20 >/dev/null 2>&1 || true
  docker exec ollama-hack-v2-db-1 psql -U ollama_hack -d ollama_hack -qc \
    "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = current_database() AND pid <> pg_backend_pid() AND state <> 'idle';" \
    >/dev/null 2>&1 || true
  "$COMPOSE" up -d backend
}

container_exists() {
  docker inspect "$1" >/dev/null 2>&1
}

health_status() {
  docker inspect "$1" --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' 2>/dev/null || echo missing
}

if ! container_exists "$GLUETUN"; then
  log "gluetun missing — bringing stack up"
  "$COMPOSE" up -d gluetun backend
  exit 0
fi

gluetun_state=$(docker inspect "$GLUETUN" --format '{{.State.Status}}' 2>/dev/null || echo missing)
if [[ "$gluetun_state" != "running" ]]; then
  log "gluetun not running (state=$gluetun_state) — starting"
  docker start "$GLUETUN" || "$COMPOSE" up -d gluetun
  rm -f "$GLUETUN_BAD_SINCE"
  exit 0
fi

gh=$(health_status "$GLUETUN")
now_sec=$(date +%s)

if [[ "$gh" == "healthy" ]]; then
  rm -f "$GLUETUN_BAD_SINCE"
elif [[ "$gh" == "unhealthy" ]]; then
  if [[ ! -f "$GLUETUN_BAD_SINCE" ]]; then
    echo "$now_sec" > "$GLUETUN_BAD_SINCE"
  fi
  bad_since=$(cat "$GLUETUN_BAD_SINCE")
  if (( now_sec - bad_since >= 300 )); then
    log "gluetun unhealthy for 5m+ — restarting gluetun (backend keeps direct probing)"
    docker restart "$GLUETUN"
    rm -f "$GLUETUN_BAD_SINCE"
    exit 0
  fi
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

b_uptime_sec=$(docker inspect "$BACKEND" --format '{{.State.StartedAt}}' 2>/dev/null | xargs -I{} date -d {} +%s 2>/dev/null || echo 0)
if [[ "$b_uptime_sec" -gt 0 && $((now_sec - b_uptime_sec)) -lt 180 ]]; then
  exit 0
fi

if ! curl -sf --max-time 8 "$HEALTH_URL" >/dev/null; then
  restart_backend "backend health check failed — restarting backend"
  exit 0
fi