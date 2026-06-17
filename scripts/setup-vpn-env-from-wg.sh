#!/bin/bash
# Build .env.vpn from /etc/wireguard/wg0.conf (Proton WireGuard) for Gluetun.
# Run on the LXC host (CT 112). Does not print private keys.
set -euo pipefail

WG_CONF="${WG_CONF:-/etc/wireguard/wg0.conf}"
OUT="${1:-.env.vpn}"

if [[ ! -f "$WG_CONF" ]]; then
  echo "Missing $WG_CONF — install Proton WireGuard config first." >&2
  exit 1
fi

priv=$(awk -F' = ' '/^PrivateKey/{print $2}' "$WG_CONF" | tr -d ' ')
addr=$(awk -F' = ' '/^Address/{print $2}' "$WG_CONF" | tr -d ' ')
pub=$(awk -F' = ' '/^PublicKey/{print $2}' "$WG_CONF" | tr -d ' ')
endpoint=$(awk -F' = ' '/^Endpoint/{print $2}' "$WG_CONF" | tr -d ' ')
ep_ip=${endpoint%%:*}
ep_port=${endpoint##*:}

if [[ -z "$priv" || -z "$addr" || -z "$pub" || -z "$ep_ip" || -z "$ep_port" ]]; then
  echo "Could not parse required fields from $WG_CONF" >&2
  exit 1
fi

cat > "$OUT" <<EOF
WIREGUARD_PRIVATE_KEY=$priv
WIREGUARD_ADDRESSES=$addr
WIREGUARD_PUBLIC_KEY=$pub
WIREGUARD_ENDPOINT_IP=$ep_ip
WIREGUARD_ENDPOINT_PORT=$ep_port
EOF

chmod 600 "$OUT"
echo "Wrote $OUT (mode 600). Endpoint: $ep_ip:$ep_port"