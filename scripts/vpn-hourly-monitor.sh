#!/bin/bash
# Extended monitor loop with optional Telegram alerts.
# Uses API checks by default; one SSH deep check per iteration when enabled.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck source=monitor-common.sh
source "$ROOT/scripts/monitor-common.sh"

monitor_load_env
LOG="${MONITOR_LOG:-/tmp/ollama-hack-hourly-monitor.log}"
INTERVAL_SEC="${INTERVAL_SEC:-300}"
RUNS="${RUNS:-12}"
MONITOR_SSH_DEEP="${MONITOR_SSH_DEEP:-1}"

monitor_acquire_lock
monitor_kill_stale_ssh

monitor_log "hourly monitor start (${RUNS} checks, every ${INTERVAL_SEC}s)"
monitor_tg "Ollama-Hack monitor started (${RUNS} checks, every ${INTERVAL_SEC}s). Will ping if API/VPN/Gluetun breaks."

alerts=0
for i in $(seq 1 "$RUNS"); do
  api_result=$(monitor_check_api)
  deep_result=""
  if [[ "$MONITOR_SSH_DEEP" == "1" ]]; then
    deep_result=$(monitor_check_deep_ssh || true)
  fi
  result=$(monitor_merge_results "$api_result" "$deep_result")
  monitor_log "run $i/$RUNS: $result"

  if [[ "$result" == ALERT* ]] || [[ "$result" == *"ALERT|"* ]]; then
    alerts=$((alerts + 1))
    monitor_tg "$(monitor_format_telegram "$result")"
    if (( alerts >= 3 )); then
      monitor_tg "Ollama-Hack: 3 consecutive issues — check CT ${CT_ID} or run: bash scripts/vpn-rollback.sh"
      break
    fi
  else
    alerts=0
  fi

  if (( i < RUNS )); then
    sleep "$INTERVAL_SEC"
  fi
done

monitor_log "hourly monitor done"