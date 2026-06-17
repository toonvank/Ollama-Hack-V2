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

docker compose -f docker-compose.yml -f docker-compose.prod.yml build frontend backend
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d

echo "Deployed. UI: http://$(hostname -I | awk '{print $1}'):3000"