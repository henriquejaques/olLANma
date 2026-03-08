# olLANma Security Analysis

Last updated: 2026-03-08

This is the canonical public security document for `olLANma`, including threat model, attack vectors, findings, and remediation status.

## 1. Executive Summary

`olLANma` has a relatively small attack surface. All high and medium findings have been remediated. The host-validation bypass (H-001) is mitigated by strict host validation (`localhost` or numeric LAN IPv4 only) plus private-range enforcement. Setup-time model auto-discovery and default-model selection are hardened with stricter LAN-only discovery, bounded/untrusted response parsing, and validated model input handling. Toolchain-related standard-library vulnerabilities were addressed by upgrading to Go `1.25` and enforcing deterministic `govulncheck` gates in CI/CD.

## 2. Threat Model

`olLANma` is a local Linux CLI that forwards requests to a remote Ollama server on a LAN.

Primary assets:

1. User configuration file (`~/.config/ollanma/config.json`).
2. User local network boundaries (preventing relay abuse).
3. User local execution environment.

Adversaries:

1. Malicious local users on the same host.
2. Malicious CLI input payloads.
3. Compromised remote Ollama target.

## 3. Attack Vectors and Mitigations

### 3.1 Local privilege escalation / config tampering

Threat: another local user reads or modifies `~/.config/ollanma/config.json`.

Mitigations:

- Config directory created with `0700`.
- Config file written with `0600`.
- Permission checks/warnings on load.

### 3.2 Memory-safety vulnerabilities

Threat: malformed/large input attempting memory corruption.

Mitigation:

- Implementation is in Go (memory-safe runtime), reducing classic buffer-overflow and UAF classes.

### 3.3 Command injection

Threat: shell metacharacters in prompts/args are interpreted as commands.

Mitigations:

- No `os/exec` usage.
- Args treated as data and encoded with `encoding/json`.

### 3.4 SSRF / arbitrary network routing

Threat: configuration abuse to route requests to unintended hosts/services.

Mitigations:

- LAN/private-IP checking via `IsPrivateIP()`.
- `ValidateHost()` rejects URL-special characters (`@`, `/`, `?`, `#`, `%`, `\`) and only permits `localhost` or numeric IPv4 hosts (H-001).
- Config is re-validated on load (M-001).
- HTTP redirects are disabled (M-002).
- Setup model auto-discovery uses strict LAN-only target policy, request URL construction via `url.URL` + `net.JoinHostPort`, disabled redirects, and a bounded timeout.

### 3.5 Malicious server responses

Threat: compromised server sends extremely large output or terminal control payloads.

Mitigations:

- Response stream capped with `io.LimitReader` (100MB).
- HTTP client timeout defaults to 300 seconds.
- Model auto-discovery response is capped to 1MB, model list is deduplicated and size-limited, and model names are validated/sanitized before display.

Residual risk:

- Streaming model output is intentionally pass-through and not terminal-sanitized.

## 4. Findings and Remediation Status

### 4.1 Critical

None.

### 4.2 High

#### H-001 Host validation and LAN-only enforcement are bypassable (Resolved 2026-03-08, Hardened 2026-03-08)

- Location: `internal/config/config.go` and `internal/client/client.go`.
- Impact: crafted hosts (for example `127.0.0.1@evil.com`) could route traffic to unintended internet hosts.
- Corrections applied:
  - `ValidateHost()` rejects hosts containing `@`, `/`, `?`, `#`, `%`, `\`, embedded ports, and non-`localhost` hostnames.
  - `IsPrivateIP()` no longer treats arbitrary hostnames as private.
  - Runtime request URLs now use `url.URL` + `net.JoinHostPort` (not string formatting).
  - Config is re-validated on `Load()` (M-001).
  - Tested: `TestValidateHost` covers 18 cases including bypass vectors.

### 4.3 Medium

#### M-001 Persisted config is not re-validated on load (Resolved 2026-03-08)

- Location: `internal/config/config.go`, `main.go`.
- Impact: manually edited/corrupt config could bypass save-time checks.
- Correction applied: `Load()` calls `ValidateConfig()` after unmarshalling. On invalid config, falls back to defaults with stderr warning.

#### M-002 HTTP redirects are followed without boundary checks (Resolved 2026-03-08)

- Location: `internal/client/client.go`.
- Impact: compromised target could redirect requests to unexpected hosts.
- Correction applied: `CheckRedirect` set to return `http.ErrUseLastResponse`, disabling all redirects. Tested: `TestForward_RejectsRedirects`.

#### M-003 Installer checksum verification is fail-open and same-channel (Resolved 2026-03-08)

- Location: `install.sh`.
- Impact: install could proceed without strong integrity verification.
- Correction applied: script now aborts on checksum download failure, empty file, missing hash, or hash mismatch.
- Residual: same-channel (binary and checksum from same GitHub release). Signed provenance (Sigstore/cosign) is a future hardening item.

#### M-004 Installer uses predictable `/tmp` paths (Resolved 2026-03-08)

- Location: `install.sh`.
- Impact: symlink/clobber risk on multi-user systems.
- Correction applied: `mktemp` for unique temp files, `trap 'rm -f ...' EXIT` for cleanup on all exit paths.

### 4.4 Low

#### L-001 Mutable GitHub Action tags in privileged release workflow (Resolved 2026-03-08)

- Location: `.github/workflows/release.yml`.
- Correction applied: Third-party GitHub Actions are now pinned by their specific commit SHAs.

#### L-002 Missing CI vulnerability scan (Resolved 2026-03-08)

- Status: resolved.
- Correction applied: CI now runs pinned `govulncheck` and fails on findings.

#### L-003 Setup model-listing and failsafe input edge cases (Resolved 2026-03-08)

- Location: `internal/config/prompt.go`.
- Historical issues:
  - numeric out-of-range selection could be silently treated as literal model name;
  - numeric model names were ambiguous with index selection;
  - model-list response parsing was unbounded;
  - terminal-unsafe model names could be printed if returned by a hostile server.
- Corrections applied:
  - added validated resolver (`resolveDefaultModelInput`) with re-prompt on invalid index;
  - prefix `=` forces numeric literal model names (for example `=1`);
  - model-list decoding is bounded (1MB), deduplicated, and model names are sanitized;
  - setup discovery now enforces stricter LAN-only/DNS-resolved host policy.
- Tests added: `TestResolveDefaultModelInput_WithDiscoveredModels`, `TestResolveDefaultModelInput_NoDiscoveredModels`, and model-list filtering tests in `internal/config/prompt_test.go`.

## 5. Tooling Results and Corrections Applied

Historical scanner state (before toolchain remediation):

- `govulncheck` on older toolchain (`go1.22.2`) reported reachable stdlib vulnerabilities.

Corrections applied:

1. Go baseline raised to `go 1.25.8` in `go.mod` (fixes GO-2026-4602 `os` Root escape and GO-2026-4601 `net/url` IPv6 parsing).
2. CI matrix updated to `1.25.x` and `stable`.
3. CI gate added/required: `go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...`.
4. Release workflow now runs the same vulnerability gate before building artifacts.
5. Config file loading/saving hardened with canonical-path and anti-symlink protections.
6. Setup model discovery and default-model selection hardened (LAN-only resolution gate, bounded parsing, and validated model input semantics).

Final validation after remediation:

- `go test -race ./...` passes.
- `go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 .` reports `No vulnerabilities found.`
- `go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...` reports `No vulnerabilities found.`
- `gosec ./...` reports `No issues found.` (`G304` resolved by hardening config file handling with strict canonical path anchoring and root-scoped/symlink-checked file access; no `#nosec` suppression remains.)

Note:

- Snap-packaged `govulncheck` may show module-detection issues (`no go.mod file`) in some environments. CI/CD and recommended local usage rely on deterministic `go run ...` execution.

## 6. CI/CD Security Gates

Both CI and release workflows run:

- `go test -race ./...`
- `go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...`

## 7. Local Security Verification

Run from repo root:

```bash
# Tests with race detector
go test -race ./...

# Static security scan (if gosec is installed)
gosec ./...

# Deterministic vuln scan (recommended)
go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...
```
