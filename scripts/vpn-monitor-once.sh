#!/bin/bash
# Single Ollama-Hack/VPN health check for systemd timer (every 5 min).
# Alerts via Telegram on failure; recovery ping when back to OK.
# Secrets: scripts/monitor.env (gitignored)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="$ROOT/scripts/monitor.env"
if [[ -f "$ENV_FILE" ]]; then
  # shellcheck source=/dev/null
  source "$ENV_FILE"
fi

: "${TELEGRAM_TOKEN:?Set TELEGRAM_TOKEN in scripts/monitor.env}"
: "${TELEGRAM_CHAT_ID:?Set TELEGRAM_CHAT_ID in scripts/monitor.env}"
: "${PVE_PASSWORD:?Set PVE_PASSWORD in scripts/monitor.env}"

PVE_HOST="${PVE_HOST:-192.168.0.158}"
CT_ID="${CT_ID:-112}"
LOG="${MONITOR_LOG:-/tmp/ollama-hack-vpn-monitor.log}"
STATE_DIR="${MONITOR_STATE_DIR:-$HOME/.cache/ollama-hack-vpn-monitor}"
STATE_FILE="$STATE_DIR/last-status"
ALERT_COOLDOWN_SEC="${ALERT_COOLDOWN_SEC:-1800}"

mkdir -p "$STATE_DIR"

tg() {
  local text="$1"
  curl -sf --max-time 15 -X POST "https://api.telegram.org/bot${TELEGRAM_TOKEN}/sendMessage" \
    -H "Content-Type: application/json" \
    -d "$(python3 -c 'import json,sys; print(json.dumps({"chat_id":sys.argv[1],"text":sys.argv[2]}))' "$TELEGRAM_CHAT_ID" "$text")" \
    >/dev/null 2>&1 || echo "[monitor] telegram send failed: $text" >>"$LOG"
}

remote() {
  sshpass -p "$PVE_PASSWORD" ssh -o StrictHostKeyChecking=no "root@${PVE_HOST}" \
    "pct exec ${CT_ID} -- bash -lc $(printf '%q' "$1")" 2>>"$LOG"
}

check_once() {
  remote '
    set -e
    fail=""
    gh=$(docker inspect ollama-hack-gluetun --format "{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}" 2>/dev/null || echo missing)
    bs=$(docker inspect ollama-hack-v2-backend-1 --format "{{.State.Status}}" 2>/dev/null || echo missing)
    health=$(curl -sf --max-time 8 http://127.0.0.1:8000/api/v2/health 2>/dev/null || echo FAIL)
    h3000=$(curl -sf --max-time 8 http://192.168.0.74:3000/api/v2/health 2>/dev/null || echo FAIL)
    vpn_mode=$(echo "$health" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('"'"'vpn'"'"',{}).get('"'"'mode'"'"','"'"''"'"'))" 2>/dev/null || echo "")
    vpn_err=$(echo "$health" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('"'"'vpn'"'"',{}).get('"'"'last_error'"'"','"'"''"'"')[:80])" 2>/dev/null || echo "")
    vpn=$(docker exec ollama-hack-gluetun wget -qO- --timeout=10 http://api.ipify.org 2>/dev/null || echo FAIL)
    crit=$(docker logs ollama-hack-v2-backend-1 2>&1 | tail -80 | grep -iE "panic|fatal|failed to connect to postgres" | tail -1 || true)
    [[ "$gh" != "healthy" ]] && fail+=" gluetun=$gh"
    [[ "$bs" != "running" ]] && fail+=" backend=$bs"
    [[ "$health" == *FAIL* ]] && fail+=" api8000=down"
    [[ "$h3000" == *FAIL* ]] && fail+=" api3000=down"
    [[ "$vpn_mode" == "direct_fallback" ]] && fail+=" vpn=direct_fallback($vpn_err)"
    [[ "${VPN_HOME_IP:-}" != "" && "$vpn" == "$VPN_HOME_IP" ]] && fail+=" vpn=home_ip_leak"
    [[ "$vpn" == "FAIL" || -z "$vpn" ]] && fail+=" vpn=egress_fail"
    [[ -n "$crit" ]] && fail+=" log=$(echo "$crit" | head -c 120)"
    if [[ -n "$fail" ]]; then
      echo "ALERT|$fail|vpn_ip=$vpn|gluetun=$gh|backend=$bs"
    else
      echo "OK|vpn_ip=$vpn|vpn_mode=$vpn_mode|gluetun=$gh|backend=$bs|api=healthy"
    fi
  '
}

read_state() {
  local key="$1" default="${2:-}"
  [[ -f "$STATE_FILE" ]] || { echo "$default"; return; }
  python3 -c '
import sys
path, key, default = sys.argv[1], sys.argv[2], sys.argv[3]
data = {}
try:
    for line in open(path):
        if "=" in line:
            k, v = line.strip().split("=", 1)
            data[k] = v
except FileNotFoundError:
    pass
print(data.get(key, default))
' "$STATE_FILE" "$key" "$default"
}

write_state() {
  local status="$1" alerting="$2" last_alert_epoch="$3"
  cat >"$STATE_FILE" <<EOF
status=$status
alerting=$alerting
last_alert_epoch=$last_alert_epoch
EOF
}

ts=$(date -Iseconds)
result=$(check_once || echo "ALERT|ssh_or_remote_fail")
echo "[$ts] $result" >>"$LOG"

prev_status=$(read_state status OK)
prev_alerting=$(read_state alerting 0)
last_alert_epoch=$(read_state last_alert_epoch 0)
now_epoch=$(date +%s)

if [[ "$result" == ALERT* ]] || [[ "$result" == *"ALERT|"* ]]; then
  should_alert=0
  if [[ "$prev_status" != ALERT* ]]; then
    should_alert=1
  elif (( now_epoch - last_alert_epoch >= ALERT_COOLDOWN_SEC )); then
    should_alert=1
  fi
  if (( should_alert )); then
    tg "Ollama-Hack ALERT: ${result#ALERT|}"
    write_state "ALERT" 1 "$now_epoch"
  else
    write_state "ALERT" 1 "$last_alert_epoch"
  fi
else
  if [[ "$prev_alerting" == "1" ]]; then
    tg "Ollama-Hack recovered: ${result#OK|}"
  fi
  write_state "OK" 0 "$last_alert_epoch"
fi