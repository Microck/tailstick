# Contributing to tailstick

Thanks for your interest in contributing. This guide covers the basics.

## Development Setup

1. **Go 1.22+** is required.
2. Clone the repository and build:
   ```bash
   go build ./...
   ```
3. Run the test suite:
   ```bash
   go test ./...
   ```

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`).
- All exported functions and types must have doc comments.
- Package-level doc comments should describe the package's purpose in 2-3 sentences.

## Pull Requests

- Open PRs against the `main` branch.
- Include a clear description of what the change does and why.
- Keep PRs focused — one concern per PR is preferred.
- Ensure `go test ./...` and `go vet ./...` pass before requesting review.

## Architecture

See [docs/architecture.md](docs/architecture.md) for the system design. Key packages:

| Package | Purpose |
|---------|---------|
| `internal/app` | Core runtime, CLI commands, and GUI server |
| `internal/tailscale` | Tailscale CLI and API client |
| `internal/state` | Persistent lease storage |
| `internal/crypto` | AES-GCM encryption for secrets |
| `internal/config` | Configuration loading and validation |
| `internal/gui` | Browser-based setup wizard |
| `internal/logging` | Structured file logging with rotation |
| `internal/model` | Core domain types |
| `internal/platform` | OS abstraction and command execution |

## Reporting Issues

Open a GitHub issue with:
- The platform and version you're running (`tailstick version`).
- Steps to reproduce.
- Expected vs actual behavior.

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
