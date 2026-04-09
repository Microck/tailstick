# Operations Guide

## Recommended Runtime Defaults

- Channel: `stable`
- Mode: `session` or `timed` for temporary access
- Timed days: `1`, `3`, or `7`
- Existing install policy: refuse unless explicit override

## Enrollment Flow

1. Operator launches CLI or GUI entrypoint explicitly.
2. TailStick loads preset and validates bounded runtime options.
3. If operator password gating is configured, TailStick validates the provided runtime password.
4. TailStick installs Tailscale if needed.
5. TailStick enrolls via `tailscale up`.
6. TailStick stores local lease state and appends USB audit entry.
7. For non-permanent leases, TailStick installs the local lease agent.

By default, TailStick reads `tailstick.config.json` from the executable directory, which matches USB deployment layout.

Local lease agent registration:

- Linux: `tailstick-agent.service` + `tailstick-agent.timer`
- Windows: `TailStickAgent-Startup` + `TailStickAgent-Periodic`

TailStick copies the running binary to a machine-local agent path and schedules that copy, so cleanup still runs after USB removal.
Cleanup uses lease-stored install snapshots and encrypted cleanup credentials first, so USB config loss does not block uninstall/device-delete workflows.

## Agent Reconciliation

Agent evaluates active leases:

- Session lease: clean on boot-id mismatch
- Timed lease: clean when `now >= expiresAt`
- Permanent lease: no automatic cleanup

Cleanup steps:

1. `tailscale down`
2. `tailscale logout`
3. Optional API delete: `DELETE /api/v2/device/{device_id}`
4. Uninstall tailscale for non-permanent leases
5. Persist status and errors in local state

If any step fails, status becomes `cleanup_failed` and retries continue on future agent passes.
If no active session/timed leases remain, TailStick self-removes local agent scheduling.

## State and Logs

Defaults:

- State: `/var/lib/tailstick/state.json` on Linux, `%ProgramData%\TailStick\state.json` on Windows
- Local log: `/var/log/tailstick.log` on Linux, `%ProgramData%\TailStick\tailstick.log` on Windows
- USB audit log: `<config_dir>/logs/tailstick-audit.ndjson`
- Secret refs: `/var/lib/tailstick/secrets/*.enc` on Linux, `%ProgramData%\TailStick\secrets\*.enc` on Windows

## Recovery Commands

Run one reconciliation cycle:

```bash
tailstick-linux-cli agent --once --config ./tailstick.config.json
```

Force cleanup for a specific lease:

```bash
tailstick-linux-cli cleanup --lease-id <lease_id> --config ./tailstick.config.json
```

## Known Constraints

- No backend management service in v1
- Linux support is Debian/Ubuntu with `systemd` only
- No stealth behavior; operations are visible in OS service/task/process state
