# GEMINI.md - olLANma

This file provides instructional context and foundational guidelines for Gemini CLI interactions within the `olLANma` project.

## Project Overview
**olLANma** is a secure, user-friendly Linux Command Line Interface (CLI) tool designed to act as a remote client for [Ollama](https://ollama.com/) instances running on a Local Area Network (LAN). 

Instead of running the LLM locally, `olLANma` allows users to query a central LAN instance as easily as if it were running on their own machine. For example, running `olLANma "hello world!"` will dispatch the prompt to the configured remote Ollama server.

## Core Features
- **Configurable Connection**: Users can easily configure the target `host`, `port`, and default `model` (e.g., via a config file in `~/.config/ollanma/` or via environment variables).
- **Seamless Prompting**: The CLI will accept prompts as arguments or through standard input (stdin) and forward them to the remote Ollama instance.
- **Easy Installation**: The tool must be easily distributable, ideally as a single, standalone binary without complex runtime dependencies.

## Recommended Tech Stack
**Go (Golang)** is the recommended language for this project.
- **Why Go?** It compiles to a single, statically-linked binary, making installation incredibly simple for users (just download and run). It has a powerful standard library for making HTTP requests (`net/http`) to interact with the Ollama API, and its memory safety features align with the project's security goals.
- *(Alternative: **Rust** could also be used for the absolute highest tier of memory safety, though Go offers a faster development cycle for HTTP-based CLIs).*

## Development Conventions & Constraints
- **Security First**: The tool must be designed to avoid open vulnerabilities. This includes safely parsing user input, handling network connections securely, and securely storing/reading configuration files without exposing sensitive local data.
- **Security Documentation**: **ALWAYS** update `docs/security_analysis.md` after fixing any vulnerability or implementing a new security control in the codebase. This document must remain the single source of truth for the project's security posture.
- **Minimal Dependencies**: To ensure easy distribution and reduce the attack surface, rely on standard library features as much as possible before introducing third-party packages.
- **Idiomatic CLI**: The tool should follow standard Linux CLI conventions (e.g., supporting `--help`, proper exit codes, and standard error handling).

## Next Steps / TODOs
- [ ] Finalize the language choice (e.g., initialize a Go module with `go mod init`).
- [ ] Define the configuration file structure (JSON, YAML, or TOML) and where it will be stored securely on the user's system.
- [ ] Implement the core HTTP client to interact with the Ollama API endpoints (e.g., `/api/generate` or `/api/chat`).
- [ ] Build the CLI argument parser to handle prompts and configuration overrides.
