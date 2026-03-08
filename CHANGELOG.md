# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

## [1.0.0] - 2026-03-08

### Added

- **First-run setup wizard** with interactive prompts for host, port, model, scheme, and timeout
- **Model auto-discovery** — setup wizard detects models available on the remote server
- **Fast Default prompting** — `olLANma "prompt"` infers `run` + default model automatically
- **Safe command forwarding** — supported commands (`run`, `generate`, `chat`, `pull`, `rm`, `cp`, `list`, `tags`, `ps`, `show`) are validated and forwarded
- **Thinking mode** — displays chain-of-thought reasoning from supported models (`show_thinking` config)
- **SSRF protection** — rejects public targets; only localhost/private/loopback/link-local IPv4 LAN targets are allowed
- **Command whitelist** — only known Ollama API commands are forwarded
- **Redirect protection** — HTTP redirects disabled to prevent redirect-based SSRF
- **Symlink protection** — rejects symlinked config files/directories
- **Atomic config writes** — temp file + rename prevents corruption
- **Response size cap** — 100 MB limit via `io.LimitReader`
- **Secure config file permissions** — directory `0700`, file `0600`, verified on every load
- **SHA256-verified installer** — `install.sh` with checksum verification for `amd64`, `arm64`, `armv7`
- **Graceful cancellation** — SIGINT/SIGTERM cleanly cancel in-flight HTTP requests
- **CI pipeline** — `go vet`, `govulncheck`, `go test -race` on every push/PR
- **Automated release pipeline** — cross-compilation + checksums on tag push
