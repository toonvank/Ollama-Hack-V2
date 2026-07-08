# Shared helpers for Ollama-Hack VPN/API monitors.
# Source from vpn-monitor-once.sh / vpn-hourly-monitor.sh (do not execute directly).

monitor_load_env() {
  local root
  root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
  local env_file="$root/scripts/monitor.env"
  if [[ -f "$env_file" ]]; then
    # shellcheck source=/dev/null
    source "$env_file"
  fi

  : "${TELEGRAM_TOKEN:?Set TELEGRAM_TOKEN in scripts/monitor.env}"
  : "${TELEGRAM_CHAT_ID:?Set TELEGRAM_CHAT_ID in scripts/monitor.env}"

  PVE_HOST="${PVE_HOST:-192.168.0.158}"
  CT_ID="${CT_ID:-112}"
  LOG="${MONITOR_LOG:-/tmp/ollama-hack-vpn-monitor.log}"
  STATE_DIR="${MONITOR_STATE_DIR:-$HOME/.cache/ollama-hack-vpn-monitor}"
  STATE_FILE="$STATE_DIR/last-status"
  ALERT_COOLDOWN_SEC="${ALERT_COOLDOWN_SEC:-1800}"
  MONITOR_API_URL="${MONITOR_API_URL:-http://192.168.0.74:3000/api/v2/health}"
  MONITOR_API_TIMEOUT_SEC="${MONITOR_API_TIMEOUT_SEC:-8}"
  MONITOR_SSH_TIMEOUT_SEC="${MONITOR_SSH_TIMEOUT_SEC:-20}"
  MONITOR_MAX_RUNTIME_SEC="${MONITOR_MAX_RUNTIME_SEC:-60}"
  MONITOR_LOCK_FILE="${MONITOR_LOCK_FILE:-/tmp/ollama-hack-vpn-monitor.lock}"
  MONITOR_SSH_DEEP="${MONITOR_SSH_DEEP:-0}"
  MONITOR_STALE_SSH_SEC="${MONITOR_STALE_SSH_SEC:-120}"

  mkdir -p "$STATE_DIR"
}

monitor_log() {
  echo "[$(date -Iseconds)] $*" >>"$LOG"
}

monitor_tg() {
  local text="$1"
  curl -sf --max-time 15 -X POST "https://api.telegram.org/bot${TELEGRAM_TOKEN}/sendMessage" \
    -H "Content-Type: application/json" \
    -d "$(python3 -c 'import json,sys; print(json.dumps({"chat_id":sys.argv[1],"text":sys.argv[2]}))' "$TELEGRAM_CHAT_ID" "$text")" \
    >/dev/null 2>&1 || monitor_log "telegram send failed: $text"
}

monitor_format_telegram() {
  python3 -c '
import re, sys

raw = sys.argv[1].strip()
if raw.startswith("ALERT|"):
    kind, body = "alert", raw[6:]
elif raw.startswith("OK|"):
    kind, body = "ok", raw[3:]
else:
    print(raw)
    raise SystemExit

parts = [p.strip() for p in body.split("|") if p.strip()]
kv = {}
issues = []
for part in parts:
    if part.startswith("vpn=direct_fallback"):
        m = re.search(r"\((.+)\)$", part)
        issues.append(("VPN proxy unreachable", m.group(1) if m else "using direct connection"))
    elif "=" in part and part.split("=", 1)[0] in {"gluetun", "backend", "vpn_ip", "vpn_mode", "api"}:
        k, v = part.split("=", 1)
        kv[k] = v
    elif "=" in part:
        k, v = part.split("=", 1)
        issues.append((k.replace("_", " "), v))
    else:
        issues.append(("issue", part))

gluetun = kv.get("gluetun", "?")
backend = kv.get("backend", "?")
vpn_ip = kv.get("vpn_ip", "?")
vpn_mode = kv.get("vpn_mode", "")

if kind == "alert":
    lines = ["⚠️ Ollama-Hack needs attention", ""]
    if issues:
        lines.append("Problem:")
        for title, detail in issues:
            if title == "VPN proxy unreachable":
                lines.append("• VPN masking is OFF — probes use your real IP")
                lines.append(f"  ({detail})")
            else:
                lines.append(f"• {title}: {detail}")
        lines.append("")
    lines.append("Still OK:")
    if backend in {"running", "up"}:
        lines.append("• API / GLM requests still work")
    if gluetun == "healthy":
        lines.append(f"• Gluetun container is healthy (VPN IP {vpn_ip})")
    lines.append("")
    lines.append("Action: fix Gluetun :8888 proxy when you can — not urgent.")
    print("\n".join(lines))
else:
    mode = vpn_mode or "proxy"
    lines = [
        "✅ Ollama-Hack recovered",
        "",
        f"• VPN mode: {mode}",
        f"• VPN egress IP: {vpn_ip}",
        f"• Gluetun: {gluetun}",
        f"• Backend: {backend}",
    ]
    print("\n".join(lines))
' "$1"
}

monitor_acquire_lock() {
  exec 9>"$MONITOR_LOCK_FILE"
  if ! flock -n 9; then
    monitor_log "skip: previous monitor run still active"
    exit 0
  fi
}

monitor_kill_stale_ssh() {
  local pid age_secs etime
  while read -r pid etime _; do
    [[ -n "$pid" ]] || continue
    age_secs=0
    if [[ "$etime" =~ ^([0-9]+):([0-5][0-9])$ ]]; then
      age_secs=$((10#${BASH_REMATCH[1]} * 60 + 10#${BASH_REMATCH[2]}))
    elif [[ "$etime" =~ ^([0-9]+)-([0-5][0-9]):([0-5][0-9])$ ]]; then
      age_secs=$((10#${BASH_REMATCH[1]} * 86400 + 10#${BASH_REMATCH[2]} * 60 + 10#${BASH_REMATCH[3]}))
    fi
    if (( age_secs >= MONITOR_STALE_SSH_SEC )); then
      kill "$pid" 2>/dev/null || true
      monitor_log "killed stale ssh pid=$pid age=${etime}s"
    fi
  done < <(ps -u "$USER" -o pid=,etime=,args= 2>/dev/null | grep "pct exec ${CT_ID}" | grep -v grep || true)
}

monitor_ssh_opts() {
  printf '%s' \
    -o BatchMode=yes \
    -o ConnectTimeout=8 \
    -o ConnectionAttempts=1 \
    -o ServerAliveInterval=5 \
    -o ServerAliveCountMax=2 \
    -o StrictHostKeyChecking=no
}

monitor_remote() {
  : "${PVE_PASSWORD:?Set PVE_PASSWORD in scripts/monitor.env for SSH deep checks}"
  timeout "$MONITOR_SSH_TIMEOUT_SEC" sshpass -p "$PVE_PASSWORD" ssh \
    $(monitor_ssh_opts) "root@${PVE_HOST}" \
    "timeout 15 pct exec ${CT_ID} -- bash -lc $(printf '%q' "$1")" 2>>"$LOG"
}

monitor_check_api() {
  local body
  body=$(curl -sf --max-time "$MONITOR_API_TIMEOUT_SEC" "$MONITOR_API_URL" 2>/dev/null) || {
    echo "ALERT|api=unreachable|url=${MONITOR_API_URL}"
    return 0
  }

  MONITOR_API_BODY="$body" python3 -c '
import json, os, sys

body = os.environ.get("MONITOR_API_BODY", "")
try:
    data = json.loads(body)
except json.JSONDecodeError:
    print("ALERT|api=invalid_json")
    raise SystemExit

fail = []
status = data.get("status", "")
vpn = data.get("vpn") or {}

if status != "healthy":
    fail.append("api_status=" + (status or "unknown"))

mode = vpn.get("mode", "")
if mode == "direct_fallback":
    err = (vpn.get("last_error") or "proxy unreachable")[:80]
    fail.append(f"vpn=direct_fallback({err})")
elif vpn.get("configured") and not vpn.get("healthy", True):
    err = (vpn.get("last_error") or "unhealthy")[:80]
    fail.append(f"vpn=unhealthy({err})")

vpn_ip = os.environ.get("VPN_HOME_IP", "").strip()
if vpn_ip and mode == "vpn":
    pass  # egress IP only available via optional SSH deep check

vpn_mode = mode or ("disabled" if not vpn.get("configured") else "unknown")

if fail:
    print("ALERT|" + " ".join(fail) + f"|vpn_mode={vpn_mode}|backend=up|gluetun=unknown|vpn_ip=unknown")
else:
    print(f"OK|vpn_mode={vpn_mode}|backend=up|gluetun=unknown|vpn_ip=unknown|api=healthy")
'
}

monitor_check_deep_ssh() {
  monitor_remote '
    set -e
    fail=""
    gh=$(docker inspect ollama-hack-gluetun --format "{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}" 2>/dev/null || echo missing)
    bs=$(docker inspect ollama-hack-v2-backend-1 --format "{{.State.Status}}" 2>/dev/null || echo missing)
    vpn=$(docker exec ollama-hack-gluetun wget -qO- --timeout=8 http://api.ipify.org 2>/dev/null || echo FAIL)
    crit=$(docker logs ollama-hack-v2-backend-1 2>&1 | tail -80 | grep -iE "panic|fatal|failed to connect to postgres" | tail -1 || true)
    [[ "$gh" != "healthy" ]] && fail+=" gluetun=$gh"
    [[ "$bs" != "running" ]] && fail+=" backend=$bs"
    [[ "${VPN_HOME_IP:-}" != "" && "$vpn" == "$VPN_HOME_IP" ]] && fail+=" vpn=home_ip_leak"
    [[ "$vpn" == "FAIL" || -z "$vpn" ]] && fail+=" vpn=egress_fail"
    [[ -n "$crit" ]] && fail+=" log=$(echo "$crit" | head -c 120)"
    if [[ -n "$fail" ]]; then
      echo "DEEP_ALERT|$fail|vpn_ip=$vpn|gluetun=$gh|backend=$bs"
    else
      echo "DEEP_OK|vpn_ip=$vpn|gluetun=$gh|backend=$bs"
    fi
  ' || echo "DEEP_ALERT|ssh_timeout_or_fail"
}

monitor_merge_results() {
  local api_result="$1"
  local deep_result="${2:-}"

  if [[ "$api_result" == ALERT* ]]; then
    if [[ -n "$deep_result" && "$deep_result" != DEEP_ALERT* ]]; then
      local deep_kv
      deep_kv="${deep_result#DEEP_OK|}"
      echo "${api_result}|${deep_kv}"
      return
    fi
    echo "$api_result"
    return
  fi

  if [[ "$deep_result" == DEEP_ALERT* ]]; then
    echo "ALERT|${deep_result#DEEP_ALERT|}"
    return
  fi

  if [[ "$deep_result" == DEEP_OK* ]]; then
    local base="${api_result#OK|}"
    local deep="${deep_result#DEEP_OK|}"
    echo "OK|${base}|${deep}"
    return
  fi

  echo "$api_result"
}

monitor_read_state() {
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

monitor_write_state() {
  local status="$1" alerting="$2" last_alert_epoch="$3"
  cat >"$STATE_FILE" <<EOF
status=$status
alerting=$alerting
last_alert_epoch=$last_alert_epoch
EOF
}

monitor_handle_result() {
  local result="$1"
  monitor_log "$result"

  local prev_status prev_alerting last_alert_epoch now_epoch should_alert
  prev_status=$(monitor_read_state status OK)
  prev_alerting=$(monitor_read_state alerting 0)
  last_alert_epoch=$(monitor_read_state last_alert_epoch 0)
  now_epoch=$(date +%s)

  if [[ "$result" == ALERT* ]] || [[ "$result" == *"ALERT|"* ]]; then
    should_alert=0
    if [[ "$prev_status" != ALERT* ]]; then
      should_alert=1
    elif (( now_epoch - last_alert_epoch >= ALERT_COOLDOWN_SEC )); then
      should_alert=1
    fi
    if (( should_alert )); then
      monitor_tg "$(monitor_format_telegram "$result")"
      monitor_write_state "ALERT" 1 "$now_epoch"
    else
      monitor_write_state "ALERT" 1 "$last_alert_epoch"
    fi
  else
    if [[ "$prev_alerting" == "1" ]]; then
      monitor_tg "$(monitor_format_telegram "$result")"
    fi
    monitor_write_state "OK" 0 "$last_alert_epoch"
  fi
}