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
HAS_VPN=0

if [[ -f /etc/wireguard/wg0.conf ]]; then
  bash scripts/setup-vpn-env-from-wg.sh .env.vpn
  COMPOSE_FILES+=(-f docker-compose.vpn.yml)
  ENV_FILE=(--env-file .env.vpn)
  HAS_VPN=1
elif [[ -f .env.vpn ]]; then
  COMPOSE_FILES+=(-f docker-compose.vpn.yml)
  ENV_FILE=(--env-file .env.vpn)
  HAS_VPN=1
fi

# Racer: background probes via VPN netns (BACKGROUND_ENDPOINT_OUTBOUND=rust).
# Default ON when VPN overlay is present — that's the point of this stack.
# Disable with RACER_COMPOSE=0.
if [[ -f docker-compose.racer.yml ]]; then
  if [[ "${RACER_COMPOSE:-}" == "0" ]]; then
    :
  elif [[ "${RACER_COMPOSE:-}" == "1" || "${HAS_VPN}" == "1" ]]; then
    COMPOSE_FILES+=(-f docker-compose.racer.yml)
  fi
fi

exec docker compose "${COMPOSE_FILES[@]}" "${ENV_FILE[@]}" "$@"
