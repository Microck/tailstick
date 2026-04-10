# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- Use supported agent install commands in E2E tests (`cfbc6a1`)
- Clear Windows live script exit state in E2E tests (`400e9f4`)
- Stop Windows agent after self-removal (`2b1b349`)
- Detach Windows agent artifact cleanup (`169089b`)
- Handle live cleanup races in E2E (`f41de17`)
- Correct live task assertions in E2E (`a313bb1`)
- Mint fresh Tailscale keys per live CI run (`169a9e2`)
- Stabilize live Windows and Linux E2E lanes (`a07d745`)
- Tighten README for operators (`807792b`)
- Fix quickstart download guidance (`daf3831`)

### Added
- Live Tailscale E2E CI workflows (`254c930`)
- Workflow test coverage for cleanup and agent reconciliation (`e300801`)

### Changed
- Poll for Windows agent artifact cleanup in E2E tests (`6234bb4`)
- Wait for Windows scheduled task state in E2E (`1ea4d9a`)
- Timeout Windows CLI phases in E2E (`a99f8a9`)
- Bound Windows live API waits in E2E (`8f595e7`)
- Update README to remove common workflows and uses (`3d512e0`)

## [v1.0.1] - 2026-04-09

### Fixed
- Allow Pages workflow to bootstrap site (`3884f6e`)

### Changed
- Simplify operator onboarding (`c1b0263`)
- Simplify quickstart copy (`ecb686b`)

## [v1.0.0] - 2026-04-09

### Added
- Cut 1.0.0 release with Pages deploy (`a647084`)
- Restyle web UI as Tailscale-style clone (`ae812fc`)
- Allow GUI host/port config and release-asset quickstart (`2abc3b9`)
- Project logo resources for GUI and Windows binaries (`a9abd1f`)

### Fixed
- Normalize Windows smoke script output matching (`650a64e`)

## [v0.1.0] - 2026-04-09

### Added
- Bootstrap tailstick with lease engine and publish-ready docs (`c45de01`)
- Release runbook and hardened `.gitignore` (`f73f935`)

[Unreleased]: https://github.com/Microck/tailstick/compare/v1.0.1...HEAD
[v1.0.1]: https://github.com/Microck/tailstick/compare/v1.0.0...v1.0.1
[v1.0.0]: https://github.com/Microck/tailstick/compare/v0.1.0...v1.0.0
[v0.1.0]: https://github.com/Microck/tailstick/releases/tag/v0.1.0
