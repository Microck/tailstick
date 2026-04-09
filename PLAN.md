# TailStick Plan

## Summary

TailStick is a USB-delivered sysadmin tool that bootstraps Tailscale connectivity on Windows and Linux machines through an explicit operator launch. It supports both CLI and GUI entrypoints, offers bounded runtime configuration on top of prebuilt presets, and can enroll machines in one of three access modes:

- Session-only
- Timed lease
- Permanent

The tool is intended for legitimate administrative use on owned or authorized machines. Installation and removal may be visible in normal OS state. The goal is reliable lifecycle control, not stealth.

## Product Goals

- Provide a consistent operator experience from a USB drive on Windows and Linux.
- Join a preconfigured Tailscale environment with minimal operator effort.
- Support both CLI and GUI entrypoints without duplicating business logic.
- Support disposable, timed, and permanent access modes.
- Remove expired leased devices from both the machine and the tailnet.
- Keep operators on safe, repeatable paths through presets and bounded runtime choices.

## Non-Goals

- No true autorun on USB insertion.
- No file-browsing or file-transfer product UX in v1.
- No broad Linux support in v1 beyond Debian/Ubuntu with `systemd`.
- No backend management service in v1.
- No attempt to hide installation or removal from normal OS visibility.
- No "nothing installed at all" lease mode if the USB is removed and cleanup must still happen later.

## Core Assumptions

- Operators can manually launch the tool and obtain elevation when needed.
- A reusable Tailscale enrollment secret is acceptable.
- Lease modes may copy cleanup-capable credentials onto the target machine.
- Unattended cleanup matters more than strong local secret protection.
- The same USB may provision multiple active leases concurrently.
- Operators may choose between a pinned stable channel and an explicit latest channel.

## Naming

### Selected Product Name

`TailStick`

### Naming Rationale

- Keeps the Tailscale association through `Tail`
- Communicates the USB form factor through `Stick`
- Reads like a product instead of a temporary codename
- Short enough for docs, logs, and binaries

## Supported Platforms

### Windows

- GUI entrypoint: `tailstick-windows-gui.exe`
- CLI entrypoint: `tailstick-windows-cli.exe`

### Linux

- GUI entrypoint: `tailstick-linux-gui`
- CLI entrypoint: `tailstick-linux-cli`

### Linux Support Scope for v1

- Debian/Ubuntu only
- `systemd` required

## UX Shape

### GUI

The GUI entrypoint starts a small local web UI, automatically opens the browser, and also prints the localhost URL for recovery or remote-operation cases.

### CLI

The CLI entrypoint provides the same workflow and state transitions without a browser dependency.

### Shared UX Rule

CLI and GUI are thin frontends over the same core workflow. They must not own separate install, lease, cleanup, or reconciliation logic.

## Architecture

### High-Level Structure

- USB assets and configuration
- Windows core
- Linux core
- Shared workflow model
- Thin CLI frontend per OS
- Thin GUI/web-launch frontend per OS

### Shared Workflow Responsibilities

- Load presets
- Accept bounded runtime options
- Detect environment details
- Request elevation when needed
- Perform install/bootstrap
- Enroll the device in Tailscale
- Persist lease metadata
- Reconcile cleanup state
- Emit logs and audit events

### OS-Specific Core Responsibilities

Windows and Linux may differ internally in service handling, secret storage, process management, and cleanup primitives. The operator flow should still feel consistent.

## Access Modes

### Session-Only

- Lease ends on machine shutdown or reboot
- Intended to use a disposable Tailscale identity
- Local access should be revoked immediately on end of session
- Tailnet cleanup should happen immediately when possible and retry until converged

### Timed Lease

Default bounded lease durations:

- 1 day
- 3 days
- 7 days

Advanced option:

- Custom integer day count with strict bounds, for example `1-30`

Timed leases use a stable identity for the lease window, then revoke local access at expiry and continue retrying machine and tailnet cleanup until complete.

### Permanent

- May leave a normal Tailscale install behind
- Does not self-delete by default
- Intended for long-lived authorized use

## Lifecycle Enforcement

### Windows

Lease modes may install a temporary local agent/service that survives launcher exit and can enforce cleanup after reboot or timeout.

### Linux

Lease modes may install a temporary local service and timer under `systemd`.

### Important Constraint

Lease modes are not "fully portable" in the strict sense. If cleanup must continue after the USB is unplugged, something local must remain installed long enough to enforce the lease and then remove itself.

## Tailscale Identity Strategy

- Each enrolled machine is a separate device
- Session-only uses disposable identity behavior
- Timed lease uses stable identity for the lease window
- Permanent uses normal persistent device behavior

### Device Naming

Default naming template:

`tsusb-{mode}-{preset}-{host}-{id}`

Example:

`tsusb-3d-opsread-finance-laptop-7q2m`

The operator may append a short suffix, but the primary naming scheme should remain deterministic.

## Presets and Runtime Configuration

### Preset-Locked Settings

- Tailnet targeting
- Enrollment/auth material
- Tags and identity class
- Control-plane cleanup credentials
- Allowed subnet list
- Cleanup mechanism
- Approved exit-node list

### Safe Runtime Settings

- Lease mode
- Lease duration from approved options
- Stable vs latest channel
- Device suffix or short label
- Exit-node selection from an approved list, if the preset allows it

### Rule

Operators should choose operational intent, not redefine security boundaries on the fly.

## Install Behavior

### Existing Tailscale Detection

If the target machine already has Tailscale installed or already belongs to some tailnet:

- Refuse by default
- Show a visible warning
- Offer an explicit advanced override for destructive replacement/adoption

This avoids breaking legitimate preexisting machine state by accident.

## Packaging and Channel Strategy

### Stable Channel

- Uses a pinned, tested Tailscale version defined in the TailStick configuration

### Latest Channel

- Explicit operator choice
- Fetches the current upstream release

### Rule

There should be no silent fallback from `latest` to `stable`. If `latest` fails, the operator should see the failure and explicitly retry with `stable`.

## Credentials and Local Protection

### USB

The USB may carry live enrollment credentials and bundled configuration.

### Target Machine

Lease modes may copy cleanup-capable credentials onto the machine so the cleanup path still works after the USB is removed.

### Security Reality

Without an operator-supplied secret at cleanup time, the machine must be able to use its local credential unattended. That means no-password mode can only protect against casual inspection, not against a sufficiently privileged attacker on the target machine.

## Logs, State, and Audit

### Machine-Local State

Each enrolled machine should keep a canonical local state record including:

- Lease ID
- Mode
- Preset
- Channel
- Hostname/device name
- Created time
- Expiry time if applicable
- Cleanup status
- Credential location reference
- Last reconciliation result

### Machine-Local Logs

Each machine should keep local operational logs and cleanup logs as plain files.

### USB Audit Trail

The USB should keep an append-only audit log of lease creation events so operators can correlate activity across multiple target machines.

### Lease ID

Every enrollment should receive a human-readable lease ID written to both USB and machine-local records.

## Failure and Recovery Model

If cleanup fails:

- Local access should be revoked immediately
- Cleanup should continue retrying in the background
- Machine-local logs/state files should record the failure clearly
- A manual `force-cleanup` path may exist, but unattended retry is the default expectation

## Scope Boundary for Networking Features

v1 should remain a client enrollment tool only.

Supported:

- Join tailnet
- Optionally select an approved exit node

Not in v1:

- Advertised subnet routes
- Exit-node hosting
- Serve/Funnel orchestration
- General portable network-appliance behavior

## Recommended Implementation Shape

### Frontends

- CLI frontend per OS
- GUI/web-launch frontend per OS

### Cores

- Windows core
- Linux core

### Shared Domain Model

Both OS cores should implement the same conceptual state machine:

- Detect
- Validate
- Elevate
- Bootstrap
- Enroll
- Persist state
- Reconcile
- Cleanup
- Remove self

The state machine should be documented and tested explicitly.

## Suggested Implementation Phases

### Phase 1

- Define preset schema
- Define lease state schema
- Define audit log schema
- Define naming and lease ID rules

### Phase 2

- Build Linux core for Debian/Ubuntu with `systemd`
- Build CLI flow first
- Implement session-only and 3-day lease lifecycle

### Phase 3

- Add Windows core
- Add Windows service-based lease enforcement
- Match CLI behavior to Linux

### Phase 4

- Add local web UI frontend
- Add GUI launchers
- Polish UX and logs

### Phase 5

- Add permanent install flow
- Add latest/stable channel selection
- Add advanced bounded custom lease duration

## Main Risks

- No-backend design pushes cleanup authority onto the target machine
- Runtime-downloaded latest versions increase field variability
- Existing Tailscale installs are easy to damage if overrides are too loose
- Windows and Linux internals will diverge if the shared workflow model is not enforced
- Custom runtime configuration can sprawl if preset boundaries are not kept strict

## Recommended Defaults

- Default channel: `stable`
- Default posture on existing install: `refuse`
- Default lease durations: `1`, `3`, `7`
- Default advanced custom duration: hidden behind an explicit advanced path
- Default scope: client connectivity only
- Default device naming: deterministic template

## Open Decisions

- Exact preset file format
- Exact machine-local state file path per OS
- Exact log file layout on USB and target machine
- Exact language/runtime choice for shared workflow implementation
- Whether Tailscale assets are bundled, pinned-download only, or mixed by profile

## Practical Conclusion

The cleanest v1 is not "magic portable Tailscale on a stick." The cleanest v1 is a disciplined admin tool:

- explicit launch
- bounded choices
- deterministic naming
- visible lifecycle
- reliable lease cleanup
- one shared workflow with two frontend styles

That shape is consistent with the requirements gathered so far and avoids the most obvious traps.
