#!/usr/bin/env bash
set -euo pipefail

TAILSTICK_BIN="${TAILSTICK_BIN:-/src/dist/tailstick-linux-cli}"
WORKDIR="/tmp/tailstick-sandbox"
FAKEBIN="/tmp/tailstick-fakebin"
CONFIG_PATH="$WORKDIR/tailstick.config.json"
STATE_PATH="$WORKDIR/state.json"
LOG_PATH="$WORKDIR/tailstick.log"
AUDIT_PATH="$WORKDIR/audit.ndjson"

mkdir -p "$WORKDIR" "$FAKEBIN"

cat > "$FAKEBIN/tailscale" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

cmd="${1:-}"
if [ "$#" -gt 0 ]; then
  shift
fi

case "$cmd" in
  version)
    echo "1.80.0"
    ;;
  update)
    echo "updated"
    ;;
  up)
    echo "up-ok"
    ;;
  status)
    if [ "${1:-}" = "--json" ]; then
      cat <<'JSON'
{"Self":{"ID":"dev-test-1","DNSName":"sandbox.node.ts.net","HostName":"sandbox"}}
JSON
    else
      echo "status"
    fi
    ;;
  down)
    echo "down-ok"
    ;;
  logout)
    echo "logout-ok"
    ;;
  *)
    echo "unknown tailscale command: $cmd" >&2
    exit 1
    ;;
esac
EOF
chmod +x "$FAKEBIN/tailscale"

cat > "$FAKEBIN/systemctl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
exit 0
EOF
chmod +x "$FAKEBIN/systemctl"

cat > "$CONFIG_PATH" <<'JSON'
{
  "stableVersion": "1.76.6",
  "defaultPreset": "ops-readonly",
  "operatorPassword": "op-pass",
  "presets": [
    {
      "id": "ops-readonly",
      "description": "sandbox integration preset",
      "authKey": "tskey-auth-fake",
      "ephemeralAuthKey": "tskey-auth-ephemeral",
      "tags": ["tag:ops-usb"],
      "acceptRoutes": true,
      "allowExitNodeSelection": true,
      "approvedExitNodes": ["100.64.0.5"],
      "install": {
        "linuxStable": ["/bin/true"],
        "linuxLatest": ["/bin/true"],
        "linuxUninstall": ["/bin/true"]
      },
      "cleanup": {
        "tailnet": "example.com",
        "deviceDeleteEnabled": false
      }
    }
  ]
}
JSON

export PATH="$FAKEBIN:$PATH"

"$TAILSTICK_BIN" run \
  --config "$CONFIG_PATH" \
  --state "$STATE_PATH" \
  --log "$LOG_PATH" \
  --audit "$AUDIT_PATH" \
  --preset ops-readonly \
  --mode timed \
  --days 3 \
  --channel stable \
  --allow-existing \
  --password op-pass \
  --exit-node 100.64.0.5

test -f /var/lib/tailstick/tailstick-agent
test -f /etc/systemd/system/tailstick-agent.service
test -f /etc/systemd/system/tailstick-agent.timer

lease_id="$(grep -o '"leaseId":[[:space:]]*"[^"]*"' "$STATE_PATH" | head -n1 | cut -d'"' -f4)"
if [ -z "$lease_id" ]; then
  echo "lease id not found in state file" >&2
  exit 1
fi

credential_ref="$(grep -o '"credentialRef":[[:space:]]*"[^"]*"' "$STATE_PATH" | head -n1 | cut -d'"' -f4)"
if [ -z "$credential_ref" ] || [ ! -f "$credential_ref" ]; then
  echo "credential ref missing or unreadable" >&2
  exit 1
fi

"$TAILSTICK_BIN" cleanup \
  --config "$CONFIG_PATH" \
  --state "$STATE_PATH" \
  --log "$LOG_PATH" \
  --audit "$AUDIT_PATH" \
  --lease-id "$lease_id"

grep -q '"status": "cleaned"' "$STATE_PATH"
test ! -f "$credential_ref"

"$TAILSTICK_BIN" agent --once \
  --config "$CONFIG_PATH" \
  --state "$STATE_PATH" \
  --log "$LOG_PATH" \
  --audit "$AUDIT_PATH"

test ! -f /var/lib/tailstick/tailstick-agent
test ! -f /etc/systemd/system/tailstick-agent.service
test ! -f /etc/systemd/system/tailstick-agent.timer

echo "linux-sandbox-e2e: PASS"
