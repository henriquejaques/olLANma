package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/henriquejaques/olLANma/internal/cli"
	"github.com/henriquejaques/olLANma/internal/client"
	"github.com/henriquejaques/olLANma/internal/config"
)

// Version is set at build time via -ldflags
var Version = "dev"

func main() {
	// Parse olLANma-specific flags, stopping at the first non-flag argument.
	// This prevents stripping flags that appear inside prompts (Gemini-4).
	skipSetup := false
	filteredArgs := []string{os.Args[0]}
	flagsDone := false

	for _, arg := range os.Args[1:] {
		if flagsDone {
			filteredArgs = append(filteredArgs, arg)
			continue
		}
		switch arg {
		case "--help", "-h":
			printUsage()
			os.Exit(0)
		case "--version", "-v":
			fmt.Printf("olLANma %s\n", Version)
			os.Exit(0)
		case "--skip-setup":
			skipSetup = true
		default:
			flagsDone = true // first non-flag arg — stop parsing flags
			filteredArgs = append(filteredArgs, arg)
		}
	}

	// 1. First-run detection: if no config file exists, run the setup wizard
	if !config.ConfigExists() && !skipSetup {
		fmt.Println("👋 Welcome to olLANma! It looks like this is your first time running it.")
		fmt.Println("Let's set up your connection to a LAN Ollama instance.")

		err := config.RunInteractivePrompt()
		if err != nil {
			fmt.Printf("Error during setup: %v\n", err)
			os.Exit(1)
		}
		fmt.Println() // breathing room before continuing
	}

	// 2. Load config (Host, Port, DefaultModel, Scheme, Timeout)
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Warning: Could not load config, using defaults. Error: %v\n", err)
	}

	// 3. Parse CLI arguments (using filtered args without --skip-setup)
	parsedCmd := cli.Parse(filteredArgs, cfg)
	if parsedCmd.Command == "" {
		printUsage()
		os.Exit(1)
	}

	// 4. Route to 'config' manager OR pass to 'client' for network request
	if parsedCmd.IsInternal {
		err := config.RunInteractivePrompt()
		if err != nil {
			fmt.Printf("Error during configuration: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// 5. Set up graceful cancellation for streaming responses
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\nInterrupted.")
		cancel()
	}()

	// 6. Pass-through to remote server
	fmt.Printf("[olLANma: Routing '%s' → %s://%s:%s]\n", parsedCmd.Command, cfg.Scheme, cfg.Host, cfg.Port)

	err = client.Forward(ctx, cfg, parsedCmd.Command, parsedCmd.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`olLANma — Remote CLI client for Ollama on your LAN

Usage:
  olLANma <prompt>              Send a prompt using the default model
  olLANma <model> <prompt>      Send a prompt using a specific model
  olLANma run <model> <prompt>  Explicit run command
  olLANma chat <model> <prompt> Chat with a model
  olLANma generate <model> <prompt> Generate text directly
  olLANma config                Configure your LAN Ollama instance
  olLANma list                  List models on the remote server
  olLANma pull <model>          Pull a model on the remote server

Supported passthrough commands:
  run, generate, chat, pull, rm, cp, list, tags, ps, show

Flags:
  --help, -h       Show this help message
  --version, -v    Show version information
  --skip-setup     Skip the first-run setup wizard (use defaults)

First Run:
  On first launch, olLANma will guide you through configuring your LAN
  connection. Run 'olLANma config' at any time to reconfigure.

Examples:
  olLANma "why is the sky blue?"
  olLANma gemma2 "tell me a joke"
  olLANma config
  olLANma list
  olLANma pull mistral`)
}
