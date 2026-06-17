#!/bin/bash
# Install systemd units on the LXC so the stack survives reboots and VPN restarts.
# Run as root inside CT 112: bash scripts/install-ollama-hack-service.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

chmod +x scripts/compose-prod.sh scripts/vpn-watchdog.sh

cat >/etc/systemd/system/ollama-hack.service <<EOF
[Unit]
Description=Ollama-Hack Docker Compose stack
After=docker.service network-online.target
Wants=network-online.target
Requires=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=$ROOT
ExecStart=$ROOT/scripts/compose-prod.sh up -d
ExecStop=$ROOT/scripts/compose-prod.sh stop
TimeoutStartSec=300

[Install]
WantedBy=multi-user.target
EOF

cat >/etc/systemd/system/ollama-hack-vpn-watchdog.service <<EOF
[Unit]
Description=Ollama-Hack VPN watchdog (sync backend after Gluetun restarts)
After=docker.service

[Service]
Type=oneshot
WorkingDirectory=$ROOT
ExecStart=$ROOT/scripts/vpn-watchdog.sh
EOF

cat >/etc/systemd/system/ollama-hack-vpn-watchdog.timer <<EOF
[Unit]
Description=Run Ollama-Hack VPN watchdog every minute

[Timer]
OnBootSec=90s
OnUnitActiveSec=60s
AccuracySec=10s

[Install]
WantedBy=timers.target
EOF

systemctl daemon-reload
systemctl enable ollama-hack.service
systemctl enable ollama-hack-vpn-watchdog.timer
systemctl start ollama-hack.service
systemctl start ollama-hack-vpn-watchdog.timer

echo "Installed:"
systemctl is-enabled ollama-hack.service
systemctl is-enabled ollama-hack-vpn-watchdog.timer
echo "Watchdog timer:"
systemctl list-timers ollama-hack-vpn-watchdog.timer --no-pager