# Local Testing Guide for olLANma

This guide explains how to spin up a pristine, reproducible environment for testing `olLANma` locally, particularly for testing the "first run" interactive installer experience without overwriting your own `~/.config/ollanma/config.json`.

We use a small, ephemeral Docker container (Alpine Linux). This ensures a clean slate on every run while still allowing the container to access your LAN/host Ollama instance.

## Why use Docker for testing?
- **Pristine State:** The container has no pre-existing configuration.
- **Safe:** You won't accidentally delete or overwrite your primary `~/.config`.
- **Network Access:** Default bridge networking gives outbound LAN access. `--network host` grants access to localhost Ollama instances.

---

## 1. Build the binary for testing

First, compile the binary on your host machine. We inject a custom version string to confirm we are running the test build.

```bash
cd ~/github/olLANma
CGO_ENABLED=0 go build -ldflags "-s -w -X main.Version=docker-test" -o olLANma .
```

## 2. Spin up the test container

Mount the compiled binary directly into the container's execution path and drop into an interactive shell.

```bash
# --rm ensures the container is deleted when you exit
# --network host ensures it can reach an Ollama instance running on localhost (127.0.0.1)
# -v mounts the binary

docker run -it --rm \
  --network host \
  -v $(pwd)/olLANma:/usr/local/bin/olLANma \
  alpine:latest \
  sh
```

## 3. Test inside the container

Once you see the `#` prompt, you are inside the fresh Alpine container. You can now act as a first-time user:

### Verify Version
```bash
olLANma --version
# Should output: olLANma docker-test
```

### Trigger First-Run Setup
Run any prompt to trigger the empty config detection:
```bash
olLANma llama3 "Are you working?"
```
*You should be prompted for host, port, scheme, and timeout.*

### Verify Configuration Artifacts
Check that the secure configuration directory and file were created with the correct permissions (`0700` and `0600`):
```bash
cat ~/.config/ollanma/config.json
ls -la ~/.config/ollanma
```

### Test Standard Commands
Passed-through standard Ollama commands should work as expected:
```bash
olLANma list
olLANma pull gemma2
```

### Test Interactive Re-configuration
```bash
olLANma config
```

## 4. Cleanup

Simply type `exit` in the container. The `--rm` flag ensures the container deletes itself instantly, leaving no trace. Your host machine's configuration remains untouched.
