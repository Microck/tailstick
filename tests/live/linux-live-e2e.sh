#!/usr/bin/env bash
set -euo pipefail

require_env() {
  local name="$1"
  if [ -z "${!name:-}" ]; then
    echo "missing required env: $name" >&2
    exit 1
  fi
}

wait_for_device_gone() {
  local device_id="$1"
  local body_file="$WORKDIR/device-api-response.json"
  local status_code

  for _ in $(seq 1 30); do
    status_code="$(curl -sS -u "${TAILSTICK_API_KEY}:" -o "$body_file" -w '%{http_code}' "https://api.tailscale.com/api/v2/device/${device_id}" || true)"
    if [ "$status_code" = "404" ]; then
      return 0
    fi
    sleep 2
  done

  echo "device ${device_id} still exists after cleanup" >&2
  cat "$body_file" >&2 || true
  return 1
}

require_env TAILSTICK_AUTH_KEY
require_env TAILSTICK_API_KEY
require_env TAILSTICK_OPERATOR_PASSWORD

if [ "$(id -u)" -ne 0 ]; then
  echo "linux live E2E must run as root inside the container" >&2
  exit 1
fi

TAILSTICK_BIN="${TAILSTICK_BIN:-/src/dist/tailstick-linux-cli}"
WORKDIR="/var/tmp/tailstick-live-e2e"
CONFIG_PATH="$WORKDIR/tailstick.config.json"
STATE_PATH="$WORKDIR/state.json"
LOG_PATH="$WORKDIR/tailstick.log"
AUDIT_PATH="$WORKDIR/audit.ndjson"
WRAPPER_DIR="$WORKDIR/bin"
TS_SOCKET="$WORKDIR/tailscaled.sock"
TS_STATE="$WORKDIR/tailscaled.state"
TS_LOG="$WORKDIR/tailscaled.log"
LEASE_ID=""
DEVICE_ID=""
TAILSCALED_PID=""

cleanup() {
  set +e

  if [ -n "$DEVICE_ID" ]; then
    curl -sS -u "${TAILSTICK_API_KEY}:" -X DELETE "https://api.tailscale.com/api/v2/device/${DEVICE_ID}" >/dev/null 2>&1 || true
  fi

  if [ -n "$TAILSCALED_PID" ]; then
    kill "$TAILSCALED_PID" >/dev/null 2>&1 || true
    wait "$TAILSCALED_PID" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

mkdir -p "$WORKDIR" "$WRAPPER_DIR"

curl -fsSL https://tailscale.com/install.sh | sh

REAL_TAILSCALE="$(command -v tailscale)"
REAL_TAILSCALED="$(command -v tailscaled)"

mkdir -p "$WORKDIR" "$WRAPPER_DIR"

cat > "$WRAPPER_DIR/tailscale" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

real="__REAL_TAILSCALE__"
socket="__TS_SOCKET__"

if [ "\${1:-}" = "version" ]; then
  exec "\$real" "\$@"
fi

exec "\$real" --socket "\$socket" "\$@"
EOF
sed -i "s|__REAL_TAILSCALE__|$REAL_TAILSCALE|g; s|__TS_SOCKET__|$TS_SOCKET|g" "$WRAPPER_DIR/tailscale"
chmod +x "$WRAPPER_DIR/tailscale"

"$REAL_TAILSCALED" --state="$TS_STATE" --socket="$TS_SOCKET" --tun=userspace-networking >"$TS_LOG" 2>&1 &
TAILSCALED_PID="$!"

for _ in $(seq 1 30); do
  if [ -S "$TS_SOCKET" ]; then
    break
  fi
  sleep 1
done

if [ ! -S "$TS_SOCKET" ]; then
  echo "tailscaled socket did not appear" >&2
  cat "$TS_LOG" >&2 || true
  exit 1
fi

cat > "$CONFIG_PATH" <<'JSON'
{
  "defaultPreset": "live-e2e-linux",
  "operatorPasswordEnv": "TAILSTICK_OPERATOR_PASSWORD",
  "presets": [
    {
      "id": "live-e2e-linux",
      "description": "live isolated Linux E2E",
      "authKeyEnv": "TAILSTICK_AUTH_KEY",
      "cleanup": {
        "tailnet": "live-e2e",
        "apiKeyEnv": "TAILSTICK_API_KEY",
        "deviceDeleteEnabled": true
      }
    }
  ]
}
JSON

export PATH="$WRAPPER_DIR:$PATH"

"$TAILSTICK_BIN" run \
  --config "$CONFIG_PATH" \
  --state "$STATE_PATH" \
  --log "$LOG_PATH" \
  --audit "$AUDIT_PATH" \
  --preset live-e2e-linux \
  --mode timed \
  --days 1 \
  --channel latest \
  --allow-existing \
  --password "$TAILSTICK_OPERATOR_PASSWORD"

LEASE_ID="$(jq -r '.records[0].leaseId' "$STATE_PATH")"
DEVICE_ID="$(jq -r '.records[0].deviceId' "$STATE_PATH")"
CREDENTIAL_REF="$(jq -r '.records[0].credentialRef' "$STATE_PATH")"
STATUS="$(jq -r '.records[0].status' "$STATE_PATH")"

test "$STATUS" = "active"
test -n "$LEASE_ID"
test -n "$DEVICE_ID"
test -f "$CREDENTIAL_REF"
test -f /var/lib/tailstick/tailstick-agent
test -f /etc/systemd/system/tailstick-agent.service
test -f /etc/systemd/system/tailstick-agent.timer
systemctl is-enabled tailstick-agent.timer >/dev/null

curl -fsS -u "${TAILSTICK_API_KEY}:" "https://api.tailscale.com/api/v2/device/${DEVICE_ID}" >/dev/null

"$TAILSTICK_BIN" cleanup \
  --config "$CONFIG_PATH" \
  --state "$STATE_PATH" \
  --log "$LOG_PATH" \
  --audit "$AUDIT_PATH" \
  --lease-id "$LEASE_ID"

STATUS_AFTER_CLEANUP="$(jq -r '.records[0].status' "$STATE_PATH")"
test "$STATUS_AFTER_CLEANUP" = "cleaned"
test ! -f "$CREDENTIAL_REF"

wait_for_device_gone "$DEVICE_ID"
DEVICE_ID=""

"$TAILSTICK_BIN" agent --once \
  --config "$CONFIG_PATH" \
  --state "$STATE_PATH" \
  --log "$LOG_PATH" \
  --audit "$AUDIT_PATH"

test ! -f /var/lib/tailstick/tailstick-agent
test ! -f /etc/systemd/system/tailstick-agent.service
test ! -f /etc/systemd/system/tailstick-agent.timer

echo "linux-live-e2e: PASS"
