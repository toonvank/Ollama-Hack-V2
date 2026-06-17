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

exec docker compose "${COMPOSE_FILES[@]}" "${ENV_FILE[@]}" "$@"