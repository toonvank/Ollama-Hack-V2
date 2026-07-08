#!/bin/bash
# Watch CT stack; optional Telegram alerts. Secrets: scripts/monitor.env (gitignored).
#   cp scripts/monitor.env.example scripts/monitor.env  # fill in, then run
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
INTERVAL_SEC="${INTERVAL_SEC:-300}"
RUNS="${RUNS:-12}"
LOG="${MONITOR_LOG:-/tmp/ollama-hack-hourly-monitor.log}"

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
    vpn_mode=$(echo "$health" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('vpn',{}).get('mode',''))" 2>/dev/null || echo "")
    vpn_err=$(echo "$health" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('vpn',{}).get('last_error','')[:80])" 2>/dev/null || echo "")
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

echo "[$(date -Iseconds)] monitor start" >>"$LOG"
tg "Ollama-Hack monitor started (${RUNS} checks, every ${INTERVAL_SEC}s). Will ping if API/VPN/Gluetun breaks."

alerts=0
for i in $(seq 1 "$RUNS"); do
  ts=$(date -Iseconds)
  result=$(check_once || echo "ALERT|ssh_or_remote_fail")
  echo "[$ts] run $i/$RUNS: $result" >>"$LOG"

  if [[ "$result" == ALERT* ]] || [[ "$result" == *"ALERT|"* ]]; then
    alerts=$((alerts + 1))
    tg "Ollama-Hack ALERT (run $i/$RUNS): ${result#ALERT|}"
    if (( alerts >= 3 )); then
      tg "Ollama-Hack: 3 consecutive issues — check CT 112 or run: bash scripts/vpn-rollback.sh"
      break
    fi
  else
    alerts=0
  fi

  if (( i < RUNS )); then
    sleep "$INTERVAL_SEC"
  fi
done

last=$(tail -1 "$LOG")
if [[ "$last" == *"OK|"* ]] && (( alerts == 0 )); then
  tg "Ollama-Hack monitor done: all checks passed. Last: ${last#*run $RUNS/$RUNS: }"
else
  tg "Ollama-Hack monitor finished with issues. See log: $LOG"
fi
echo "[$(date -Iseconds)] monitor end alerts=$alerts" >>"$LOG"