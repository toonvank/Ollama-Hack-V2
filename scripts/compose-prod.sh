#!/bin/bash
# Build docker compose command for Proxmox production (with optional VPN overlay).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ ! -f docker-compose.yml ]]; then
  cp docker-compose.dev.yml docker-compose.yml
fi

COMPOSE_FILES=(-f docker-compose.yml -f docker-compose.prod.yml)
ENV_FILE=()

if [[ -f /etc/wireguard/wg0.conf ]]; then
  bash scripts/setup-vpn-env-from-wg.sh .env.vpn
  COMPOSE_FILES+=(-f docker-compose.vpn.yml)
  ENV_FILE=(--env-file .env.vpn)
elif [[ -f .env.vpn ]]; then
  COMPOSE_FILES+=(-f docker-compose.vpn.yml)
  ENV_FILE=(--env-file .env.vpn)
fi

# Optional Rust sidecar (Phase 1 relay). Enable with RACER_COMPOSE=1.
#   RACER_COMPOSE=1 bash scripts/compose-prod.sh up -d --build racer backend
if [[ "${RACER_COMPOSE:-0}" == "1" ]]; then
  COMPOSE_FILES+=(-f docker-compose.racer.yml)
fi

exec docker compose "${COMPOSE_FILES[@]}" "${ENV_FILE[@]}" "$@"