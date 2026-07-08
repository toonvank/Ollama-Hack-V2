#!/bin/bash
# Install user systemd timer: Ollama-Hack VPN monitor every 5 minutes.
# Run on your workstation (needs scripts/monitor.env + sshpass).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
USER_NAME="${USER:-$(whoami)}"
UNIT_DIR="$HOME/.config/systemd/user"
SERVICE_NAME=ollama-hack-vpn-monitor

if [[ ! -f "$ROOT/scripts/monitor.env" ]]; then
  echo "Missing $ROOT/scripts/monitor.env — copy scripts/monitor.env.example and fill in secrets." >&2
  exit 1
fi

chmod +x "$ROOT/scripts/vpn-monitor-once.sh" "$ROOT/scripts/vpn-hourly-monitor.sh" "$ROOT/scripts/monitor-common.sh"

mkdir -p "$UNIT_DIR"

cat >"$UNIT_DIR/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=Ollama-Hack VPN/API health check (single run)
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
WorkingDirectory=$ROOT
TimeoutStartSec=90
TimeoutStopSec=15
ExecStart=/usr/bin/timeout 90 $ROOT/scripts/vpn-monitor-once.sh
EOF

cat >"$UNIT_DIR/${SERVICE_NAME}.timer" <<EOF
[Unit]
Description=Run Ollama-Hack VPN monitor every 5 minutes

[Timer]
OnBootSec=2min
OnUnitActiveSec=5min
AccuracySec=30s
Persistent=true

[Install]
WantedBy=timers.target
EOF

systemctl --user daemon-reload
systemctl --user enable --now "${SERVICE_NAME}.timer"

if ! loginctl show-user "$USER_NAME" -p Linger 2>/dev/null | grep -q 'Linger=yes'; then
  echo "Enabling linger so the timer runs without an active login session..."
  sudo loginctl enable-linger "$USER_NAME"
fi

echo "Installed user timer:"
systemctl --user list-timers "${SERVICE_NAME}.timer" --no-pager
echo ""
echo "Test run:"
systemctl --user start "${SERVICE_NAME}.service"
echo "Log: /tmp/ollama-hack-vpn-monitor.log"