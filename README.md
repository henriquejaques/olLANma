# 🦙 olLANma

[![CI](https://github.com/henriquejaques/olLANma/actions/workflows/ci.yml/badge.svg)](https://github.com/henriquejaques/olLANma/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/henriquejaques/olLANma)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/henriquejaques/olLANma)](https://github.com/henriquejaques/olLANma/releases/latest)
[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/henriquejaques)

**A secure, zero-dependency CLI client for [Ollama](https://ollama.com/) instances on your LAN.**

Run LLMs on a powerful server in your home lab and query them from any machine on the network — as easily as if Ollama were running locally.

```bash
olLANma "why is the sky blue?"
```

---

## Why olLANma?

Not every machine on your network can (or should) run a large language model. Maybe you have a beefy GPU server in a closet, a Raspberry Pi on your desk, or a thin laptop you use on the couch. **olLANma** bridges the gap:

- **One server, many clients** — Install Ollama on your most powerful box, then query it from anywhere on your LAN.
- **Zero friction** — No Docker, no port-forwarding, no API keys. Just a single binary and a 30-second guided setup.
- **Security built-in** — LAN-only by default, with SSRF protection, input validation, and hardened config file permissions.

---

## Table of Contents

- [Value Proposition](#value-proposition)
- [Features](#features)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [First Run](#first-run)
- [Usage](#usage)
- [Configuration](#configuration)
- [Security](#security)
- [Architecture](#architecture)
- [Contributing](#contributing)
- [License](#license)

---

## Value Proposition

`olLANma` is built for people who want a **terminal-first**, **low-ops**, and **safer-by-default** way to use a shared Ollama server on a local network.

Looking for audience-specific examples? Jump to [For Technical Teams](#for-technical-teams) or [For Non-Technical Users](#for-non-technical-users).

### For Technical Teams

- **Infra/Platform teams** exposing one internal Ollama node to multiple developer workstations without shipping custom curl scripts.
- **SRE/DevOps workflows** that need model operations (`list`, `pull`, `ps`, `rm`) from SSH sessions and automation scripts.
- **Security-conscious engineering orgs** that want CLI access with built-in LAN target restrictions and safer defaults.

### For Non-Technical Users

- **Writers/researchers on lightweight laptops** who need local AI speed from a shared office/home server without learning APIs.
- **Small business teams** that want a private, local AI assistant in terminal workflows with a guided setup and minimal maintenance.
- **Educators/students in labs** where one configured server can support many client machines quickly.

### Real Problems It Solves

| Problem | How olLANma helps |
| --- | --- |
| **You want remote Ollama to feel local in terminal** | Keeps a CLI-native workflow with fast defaults: `olLANma "prompt"` just works without hand-crafting JSON requests. |
| **You need to use LLMs over SSH/headless systems** | Single binary, no web app stack, no browser session needed. Works cleanly in tmux, remote shells, and low-resource machines. |
| **You want safer defaults on a LAN** | Enforces host/scheme/port validation, private-network target restrictions, redirect blocking, and a command whitelist. |
| **You need quick onboarding for non-API users** | First-run wizard + model auto-discovery + stored defaults reduce setup friction to minutes. |
| **You need shell automation around local inference** | Script-friendly CLI behavior with predictable commands, timeouts, and graceful cancellation for CI/jobs/scripts. |

### Why Not Just Direct API or OpenWebUI?

| If your priority is... | Direct Ollama API | OpenWebUI | olLANma |
| --- | --- | --- | --- |
| **Raw flexibility at HTTP level** | Best fit | Not primary focus | Uses API under the hood, but intentionally constrained |
| **Rich graphical multi-user chat interface** | Not ideal | Best fit | Not the goal |
| **Fast terminal workflows and scripting** | Possible, but more boilerplate | Not terminal-native | Best fit |
| **Minimal deployment/ops overhead** | Moderate (you own request tooling) | Higher (web app to run/maintain) | Low (single binary) |
| **Secure-by-default CLI bridge to LAN Ollama** | DIY guardrails | Different web-app threat model | Core design goal |

In short: **use OpenWebUI for collaborative browser UX, use raw API for full protocol control, and use olLANma when you want remote Ollama with local CLI ergonomics and built-in safety defaults.**

---

## Features

|     | Feature                     | Description                                                                                                                         |
| --- | --------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| 🧙  | **First-Run Wizard**        | Guided setup on first launch — no manual config editing needed                                                                      |
| 🔍  | **Model Auto-Discovery**    | The setup wizard automatically detects models available on your server                                                              |
| ⚡  | **Fast Defaults**           | Just type a prompt — olLANma infers `run` + your default model                                                                      |
| 🔀  | **Safe Command Forwarding** | Supported remote commands (`run`, `generate`, `chat`, `list`, `tags`, `pull`, `rm`, `show`, `ps`, `cp`) are validated and forwarded |
| 🧠  | **Thinking Mode**           | Optionally display model-provided thinking output (when available)                                                                  |
| 📦  | **Single Binary**           | Zero runtime dependencies — download and run                                                                                        |
| 🔒  | **Security First**          | SSRF protection, input validation, secure config, no shell execution                                                                |
| 🏗️  | **Zero Dependencies**       | Built entirely on the Go standard library — no supply chain attack surface                                                          |

---

## Quick Start

```bash
# Install
curl -fsSL https://raw.githubusercontent.com/henriquejaques/olLANma/main/install.sh | sh

# Use (first run triggers the setup wizard automatically)
olLANma "explain quantum entanglement like I'm five"
```

---

## Installation

### Option 1: One-Line Installer (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/henriquejaques/olLANma/main/install.sh | sh
```

> Supports `amd64`, `arm64`, and `armv7`. The script verifies the download with **SHA256 checksums** and uses `sudo` to install to `/usr/local/bin/`.

### Option 2: Manual Download

```bash
# Download the latest binary (replace ARCH with amd64, arm64, or armv7)
curl -L -o olLANma https://github.com/henriquejaques/olLANma/releases/latest/download/olLANma-linux-ARCH

# Make it executable & move it to your PATH
chmod +x olLANma
sudo mv olLANma /usr/local/bin/
```

### Option 3: Build From Source

```bash
go install github.com/henriquejaques/olLANma@latest
```

### Troubleshooting (Alpine Linux)

If `olLANma` autocompletes but running it returns `sh: olLANma: not found`, the file is installed but your system can't load its runtime linker.

On Alpine, install glibc compatibility and retry:

```bash
apk add --no-cache libc6-compat
```

---

## First Run

On first launch, olLANma detects that no configuration exists and walks you through setup:

```
$ olLANma "hello!"
👋 Welcome to olLANma! It looks like this is your first time running it.
Let's set up your connection to a LAN Ollama instance.

=== olLANma Configuration Setup ===
Host (e.g., 192.168.1.50 or 127.0.0.1) [127.0.0.1]: 192.168.1.50
Port (usually 11434) [11434]:
Scheme (http or https) [http]:
Timeout in seconds (1-3600) [300]:
Show model thinking output (y/n) [y]:

Fetching available models from 192.168.1.50...
Available models on this server:
  1) llama3
  2) gemma2
Default Model (e.g., llama3, mistral) [llama3]:

Configuration successfully saved to ~/.config/ollanma/

[olLANma: Routing 'run' → http://192.168.1.50:11434]
Hello! How can I help you today?
```

After setup, your original prompt runs immediately — no need to retype anything.  
Run `olLANma config` at any time to reconfigure.

---

## Usage

### Prompting

```bash
# Uses your default model
olLANma "why is the sky blue?"

# Specify a model explicitly
olLANma gemma2 "tell me a joke"

# Explicit 'run' command (same as above)
olLANma run llama3 "explain quantum computing"

# Chat API shortcut
olLANma chat llama3 "give me 3 startup ideas"
```

### Ollama Passthrough Commands

Supported commands are forwarded to the remote server:

```bash
olLANma run llama3 "hello"       # Generate via run
olLANma generate llama3 "hello"  # Generate via generate API
olLANma chat llama3 "hello"      # Chat API
olLANma list                    # List available models
olLANma tags                    # Alias for list/tags endpoint
olLANma pull mistral            # Pull a model
olLANma rm old-model            # Remove a model
olLANma show llama3             # Show model details
olLANma ps                      # Show running models
olLANma cp llama3 my-llama3     # Copy a model
```

Unsupported commands or unexpected extra arguments fail with a clear validation error instead of being silently ignored.

### olLANma Flags

| Flag              | Description                              |
| ----------------- | ---------------------------------------- |
| `--help`, `-h`    | Show help message                        |
| `--version`, `-v` | Show version information                 |
| `--skip-setup`    | Skip the first-run wizard (use defaults) |

---

## Configuration

Settings are stored in `~/.config/ollanma/config.json` with secure `0600` permissions:

```json
{
	"host": "192.168.1.50",
	"port": "11434",
	"default_model": "llama3",
	"scheme": "http",
	"timeout": 300,
	"show_thinking": true
}
```

| Setting         | Description                                           | Default     |
| --------------- | ----------------------------------------------------- | ----------- |
| `host`          | LAN IPv4 address or `localhost` of your Ollama server | `127.0.0.1` |
| `port`          | Port number (1–65535)                                 | `11434`     |
| `default_model` | Model used when none is specified                     | `llama3`    |
| `scheme`        | Protocol: `http` or `https`                           | `http`      |
| `timeout`       | HTTP timeout in seconds (1–3600)                      | `300`       |
| `show_thinking` | Display model-provided thinking output when available | `true`      |

To reconfigure at any time:

```bash
olLANma config
```

---

## Security

olLANma is designed with a **security-first** philosophy. Zero third-party dependencies means zero supply chain attack surface.

| Protection                       | Implementation                                                                                         |
| -------------------------------- | ------------------------------------------------------------------------------------------------------ |
| **Memory safety**                | Written in Go — garbage collected, immune to buffer overflows and use-after-free                       |
| **JSON injection prevention**    | All payloads built via `encoding/json.Marshal`, never string interpolation                             |
| **Command injection prevention** | `os/exec` is never used — all input is treated as literal string data                                  |
| **Input validation**             | Host, port, scheme, timeout, and model are validated before saving                                     |
| **SSRF protection**              | Rejects public targets by default; allows localhost and private/loopback/link-local IPv4 LAN addresses |
| **Redirect protection**          | HTTP redirects are disabled entirely, preventing redirect-based SSRF                                   |
| **Command whitelist**            | Only known Ollama API commands are forwarded — blocks path traversal attacks                           |
| **Response size limit**          | Responses capped at **100 MB** via `io.LimitReader` (prevents infinite streaming)                      |
| **HTTP timeout**                 | Configurable timeout (default 300s) prevents indefinite hangs                                          |
| **Config file security**         | Config dir: `0700`, config file: `0600` — permissions verified on every load                           |
| **Symlink protection**           | Rejects symlinked config files/directories to prevent path redirection                                 |
| **Atomic writes**                | Config saves use temp file + rename to prevent corruption                                              |
| **Integrity verification**       | Install script verifies SHA256 checksums before installation                                           |
| **Graceful cancellation**        | SIGINT/SIGTERM handled cleanly, cancelling in-flight HTTP requests                                     |

### Verify Locally

```bash
# Run tests with race detector
go test -race ./...

# Static security scan (requires gosec)
gosec ./...

# Vulnerability scan (pinned version — same as CI)
go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...
```

For the full threat model, attack vectors, and mitigations, see [docs/security_analysis.md](docs/security_analysis.md).

---

## Architecture

```
olLANma/
├── main.go                  # Entry point: flag parsing, signal handling, routing
├── internal/
│   ├── cli/
│   │   └── parser.go        # Smart CLI parser with "Fast Default" behavior
│   ├── client/
│   │   └── client.go        # HTTP client — command routing, streaming, security controls
│   └── config/
│       ├── config.go        # Config struct, validation, load/save with security checks
│       └── prompt.go        # Interactive setup wizard with model auto-discovery
├── docs/
│   └── security_analysis.md # Threat model, findings, and remediation status
├── install.sh               # One-line installer with SHA256 verification
├── Dockerfile.test          # E2E test for install.sh in a clean container
├── test_install.sh          # Install verification script (runs inside Docker)
└── .github/
    ├── workflows/
    │   ├── ci.yml           # CI: vet, govulncheck, build, test (with race detector)
    │   └── release.yml      # Release: cross-compile + checksums on tag push
    └── ISSUE_TEMPLATE/      # Bug report & feature request templates
```

---

## Contributing

Contributions are welcome! Here's how to get started:

1. **Fork** the repository
2. **Create a branch** for your feature or fix (`git checkout -b feature/my-feature`)
3. **Make your changes** — follow the existing code style and keep dependencies to zero
4. **Run the tests** — `go test -race ./...`
5. **Open a Pull Request** with a clear description of what and why

Please review [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) before participating.

> [!IMPORTANT]
> If your change touches security controls, please update [docs/security_analysis.md](docs/security_analysis.md) as well.

---

## License

This project is licensed under the **MIT License** — see the [LICENSE](LICENSE) file for details.
