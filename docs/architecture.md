# Architecture

## Core Principle

TailStick has one workflow engine with two frontend styles:

- CLI entrypoint
- Local web UI entrypoint

Both frontends call the same state machine and the same tailscale integration logic.

## State Machine

1. Detect environment
2. Validate preset and runtime options
3. Enforce elevation requirement for privileged operations
4. Ensure tailscale install and stable pinning when channel is `stable`
5. Enroll (`tailscale up`)
6. Persist lease state and audit event
7. Install lease agent for non-permanent modes
8. Reconcile cleanup on timer/startup
9. Retry until cleanup convergence
10. Remove self when no active managed leases remain

## Packages

- `internal/app`: workflow engine, command handlers, agent routines
- `internal/gui`: local web launcher and API
- `internal/config`: config loading and validation
- `internal/state`: local state and USB audit persistence
- `internal/tailscale`: tailscale CLI + control-plane delete API integration
- `internal/platform`: OS detection, runtime paths, command execution
- `internal/crypto`: local secret encryption/decryption helpers
- `internal/logging`: local structured logging

## Lease Modes

- Session lease:
  - active for current boot session
  - cleanup on boot mismatch
- Timed lease:
  - active until `expiresAt`
  - cleanup once expired, then retry on failure
- Permanent:
  - no automatic cleanup

Lease records persist install-command snapshots and encrypted cleanup credentials so cleanup can proceed without requiring the original USB config file.

## Reconciliation

Agent actions:

1. `tailscale down`
2. `tailscale logout`
3. Optional API `DELETE /api/v2/device/{device_id}`
4. Uninstall tailscale for non-permanent leases

If any step fails, the record moves to `cleanup_failed` and retries continue in future iterations.
If no managed leases remain, the agent removes its own schedule/service registration.

The lease agent runs from a machine-local binary copy (`/var/lib/tailstick/tailstick-agent` on Linux, `%ProgramData%\TailStick\tailstick-agent.exe` on Windows) so reconciliation remains available when USB media is absent.
