#!/bin/bash
# Fast rollback: restore pre-VPN compose overlay and recreate backend on app-network.
# Run inside CT 112: bash scripts/vpn-rollback.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BACKEND=ollama-hack-v2-backend-1
COMPOSE="$ROOT/scripts/compose-prod.sh"

log() { echo "[vpn-rollback $(date -Iseconds)] $*"; }

if [[ -f docker-compose.vpn.yml.rollback-snapshot ]]; then
  log "restoring docker-compose.vpn.yml from snapshot"
  cp docker-compose.vpn.yml.rollback-snapshot docker-compose.vpn.yml
else
  log "no snapshot found — stripping HTTP_PROXY from running backend"
  docker update --env-rm HTTP_PROXY --env-rm HTTPS_PROXY --env-rm NO_PROXY "$BACKEND" 2>/dev/null || true
  log "recreating backend without proxy overlay env"
  "$COMPOSE" up -d --force-recreate backend
  exit 0
fi

log "recreating backend without VPN proxy"
"$COMPOSE" up -d --force-recreate backend

if curl -sf --max-time 8 http://127.0.0.1:8000/api/v2/health >/dev/null; then
  log "rollback OK — API healthy, outbound direct (no VPN proxy)"
else
  log "WARNING: API health check failed after rollback"
  exit 1
fi