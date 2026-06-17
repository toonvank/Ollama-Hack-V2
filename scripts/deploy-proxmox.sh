#!/bin/bash
# Deploy Ollama-Hack on Proxmox LXC (e.g. CT 112 ollama-hack-v2-docker-lxe).
# Run inside the LXC: bash scripts/deploy-proxmox.sh
set -euo pipefail

cd "$(dirname "$0")/.."

chmod +x scripts/compose-prod.sh scripts/vpn-watchdog.sh scripts/install-ollama-hack-service.sh

git pull origin main

bash scripts/compose-prod.sh build frontend backend
bash scripts/compose-prod.sh up -d

if [[ -f docker-compose.vpn.yml ]] && { [[ -f /etc/wireguard/wg0.conf ]] || [[ -f .env.vpn ]]; }; then
  echo "VPN: installing systemd boot + watchdog (survives reboots and Gluetun restarts)"
  bash scripts/install-ollama-hack-service.sh
else
  echo "VPN: skipped (no wg0.conf or .env.vpn)"
fi

echo "Deployed. UI: http://$(hostname -I | awk '{print $1}'):3000"