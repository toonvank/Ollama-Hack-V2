#!/bin/bash
# Deploy Ollama-Hack on Proxmox LXC (e.g. CT 112 ollama-hack-v2-docker-lxe).
# Run inside the LXC: bash scripts/deploy-proxmox.sh
set -euo pipefail

cd "$(dirname "$0")/.."

if [[ ! -f docker-compose.yml ]]; then
  cp docker-compose.dev.yml docker-compose.yml
  echo "Created docker-compose.yml from docker-compose.dev.yml — set real secrets before production use."
fi

git pull origin main

COMPOSE_FILES=(-f docker-compose.yml -f docker-compose.prod.yml)
ENV_FILE=()

if [[ -f /etc/wireguard/wg0.conf ]]; then
  bash scripts/setup-vpn-env-from-wg.sh .env.vpn
  COMPOSE_FILES+=(-f docker-compose.vpn.yml)
  ENV_FILE=(--env-file .env.vpn)
  echo "VPN: Gluetun overlay enabled (.env.vpn from wg0.conf)"
elif [[ -f .env.vpn ]]; then
  COMPOSE_FILES+=(-f docker-compose.vpn.yml)
  ENV_FILE=(--env-file .env.vpn)
  echo "VPN: Gluetun overlay enabled (.env.vpn)"
else
  echo "VPN: skipped (no wg0.conf or .env.vpn)"
fi

docker compose "${COMPOSE_FILES[@]}" "${ENV_FILE[@]}" build frontend backend
docker compose "${COMPOSE_FILES[@]}" "${ENV_FILE[@]}" up -d

echo "Deployed. UI: http://$(hostname -I | awk '{print $1}'):3000"