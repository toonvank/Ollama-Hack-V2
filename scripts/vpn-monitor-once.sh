#!/bin/bash
# Single Ollama-Hack/VPN health check for systemd timer (every 5 min).
# API-first (no SSH by default) — fast, cannot wedge the LXC.
# Optional SSH deep check: MONITOR_SSH_DEEP=1 in monitor.env
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck source=monitor-common.sh
source "$ROOT/scripts/monitor-common.sh"

monitor_load_env
monitor_acquire_lock
monitor_kill_stale_ssh

api_result=$(monitor_check_api)
deep_result=""
if [[ "$MONITOR_SSH_DEEP" == "1" ]]; then
  deep_result=$(monitor_check_deep_ssh || true)
fi

monitor_handle_result "$(monitor_merge_results "$api_result" "$deep_result")"