package config

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	modelDiscoveryTimeout        = 3 * time.Second
	modelResolveTimeout          = 1 * time.Second
	maxModelListResponseBytes    = 1 * 1024 * 1024 // 1MB cap for untrusted model list responses
	maxDiscoveredModels          = 200
	maxDiscoveredModelNameLength = 256
)

// RunInteractivePrompt guides the user through setting up their LAN instance details
func RunInteractivePrompt() error {
	fmt.Println("=== olLANma Configuration Setup ===")

	// Load existing first to use as defaults
	cfg, err := Load()
	if err != nil {
		fmt.Printf("Note: creating first-time configuration (%v)\n", err)
		cfg = DefaultConfig()
	}

	reader := bufio.NewReader(os.Stdin)

	// promptField asks for a value and validates it with the provided validator.
	// On validation failure, it re-prompts the user.
	promptField := func(fieldLabel string, currentValue string, validate func(string) error) string {
		for {
			fmt.Printf("%s [%s]: ", fieldLabel, currentValue)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input == "" {
				return currentValue // Keep existing/default if empty
			}
			if validate != nil {
				if err := validate(input); err != nil {
					fmt.Printf("  ⚠ Invalid input: %v. Please try again.\n", err)
					continue
				}
			}
			return input
		}
	}

	cfg.Host = promptField("Host (e.g., 192.168.1.50 or 127.0.0.1)", cfg.Host, ValidateHost)
	cfg.Port = promptField("Port (usually 11434)", cfg.Port, ValidatePort)

	schemeValidator := func(s string) error {
		return ValidateScheme(s)
	}
	cfg.Scheme = promptField("Scheme (http or https)", cfg.Scheme, schemeValidator)

	// Timeout — needs special handling since it's an int
	timeoutStr := promptField(
		"Timeout in seconds (1-3600)",
		strconv.Itoa(cfg.Timeout),
		func(s string) error {
			v, err := strconv.Atoi(s)
			if err != nil {
				return fmt.Errorf("must be a number, got %q", s)
			}
			return ValidateTimeout(v)
		},
	)
	cfg.Timeout, _ = strconv.Atoi(timeoutStr)

	showThinkingDefault := "n"
	if cfg.ShowThinking {
		showThinkingDefault = "y"
	}
	showThinkingStr := promptField(
		"Show model thinking output (y/n)",
		showThinkingDefault,
		func(s string) error {
			s = strings.TrimSpace(strings.ToLower(s))
			if s != "y" && s != "yes" && s != "n" && s != "no" {
				return fmt.Errorf("must be 'y' or 'n'")
			}
			return nil
		},
	)
	showThinkingStr = strings.TrimSpace(strings.ToLower(showThinkingStr))
	cfg.ShowThinking = showThinkingStr == "y" || showThinkingStr == "yes"

	// Attempt to fetch available models from the configured host to help the user
	fmt.Printf("\nFetching available models from %s...\n", cfg.Host)
	models := fetchAvailableModels(cfg)
	if len(models) > 0 {
		fmt.Println("Available models on this server:")
		for i, m := range models {
			fmt.Printf("  %d) %s\n", i+1, m)
		}

		// If the current default model isn't in the list, default to the first one safely
		modelFound := false
		for _, m := range models {
			if m == cfg.DefaultModel {
				modelFound = true
				break
			}
		}
		if !modelFound && cfg.DefaultModel == "llama3" { // Only overwrite if it was the hardcoded default
			cfg.DefaultModel = models[0]
		}
	} else {
		fmt.Println("  (No models found on the server, or unable to reach it to auto-discover models)")
		fmt.Println("  Once setup is complete, you can download models by running:")
		fmt.Println("    olLANma pull llama3")
		fmt.Println("    or")
		fmt.Println("    olLANma pull gemma2")
	}

	rawModel := promptField("Default Model (e.g., llama3, mistral)", cfg.DefaultModel, nil)
	for {
		selected, ok, errMsg := resolveDefaultModelInput(rawModel, models)
		if ok {
			cfg.DefaultModel = selected
			break
		}
		if len(models) > 0 {
			fmt.Printf("  ⚠ Invalid model selection: %s. Choose 1-%d, type a model name, or prefix '=' for numeric literal names.\n", errMsg, len(models))
		} else {
			fmt.Printf("  ⚠ Invalid model selection: %s. Please try again.\n", errMsg)
		}
		rawModel = promptField("Default Model (e.g., llama3, mistral)", cfg.DefaultModel, nil)
	}

	// Save back to disk
	err = Save(cfg)
	if err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("\nConfiguration successfully saved to ~/.config/%s/\n", configDirName)
	return nil
}

func isPrivateLikeIP(ip net.IP) bool {
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast())
}

// hostAllowedForModelDiscovery enforces LAN-only targets for model discovery.
// It keeps DNS-resolution checks as defense-in-depth if host validation ever relaxes.
func hostAllowedForModelDiscovery(host string) bool {
	if host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return isPrivateLikeIP(ip)
	}

	ctx, cancel := context.WithTimeout(context.Background(), modelResolveTimeout)
	defer cancel()

	resolved, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil || len(resolved) == 0 {
		return false
	}
	for _, addr := range resolved {
		if !isPrivateLikeIP(addr.IP) {
			return false
		}
	}
	return true
}

func isSafeModelName(name string) bool {
	if name == "" || len(name) > maxDiscoveredModelNameLength || !utf8.ValidString(name) {
		return false
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

// resolveDefaultModelInput interprets user input for default-model selection.
// If models are listed, numeric input selects by index; prefix "=" forces literal.
func resolveDefaultModelInput(raw string, models []string) (model string, ok bool, errMsg string) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false, "model cannot be empty"
	}

	forceLiteral := strings.HasPrefix(value, "=")
	if forceLiteral {
		value = strings.TrimSpace(strings.TrimPrefix(value, "="))
		if value == "" {
			return "", false, "literal model name after '=' cannot be empty"
		}
	}

	if len(models) > 0 && !forceLiteral {
		if num, err := strconv.Atoi(value); err == nil {
			if num < 1 || num > len(models) {
				return "", false, fmt.Sprintf("selection %d is out of range (1-%d)", num, len(models))
			}
			return models[num-1], true, ""
		}
	}

	if !isSafeModelName(value) {
		return "", false, "model contains control characters, invalid UTF-8, or exceeds max length"
	}
	return value, true, ""
}

// fetchAvailableModels attempts to query the Ollama /api/tags endpoint to list available models.
func fetchAvailableModels(cfg Config) []string {
	// Re-validate and apply stricter safety policy before network call.
	if err := ValidateHost(cfg.Host); err != nil {
		return nil
	}
	if err := ValidatePort(cfg.Port); err != nil {
		return nil
	}
	if err := ValidateScheme(cfg.Scheme); err != nil {
		return nil
	}
	if !hostAllowedForModelDiscovery(cfg.Host) {
		return nil
	}

	reqURL := (&url.URL{
		Scheme: cfg.Scheme,
		Host:   net.JoinHostPort(cfg.Host, cfg.Port),
		Path:   "/api/tags",
	}).String()

	client := &http.Client{
		Timeout: modelDiscoveryTimeout, // Don't hang the prompt for too long
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), modelDiscoveryTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	limited := io.LimitReader(resp.Body, maxModelListResponseBytes)
	if err := json.NewDecoder(limited).Decode(&result); err != nil {
		return nil
	}

	names := make([]string, 0, len(result.Models))
	seen := make(map[string]struct{}, len(result.Models))
	for _, m := range result.Models {
		name := strings.TrimSpace(m.Name)
		if !isSafeModelName(name) {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
		if len(names) >= maxDiscoveredModels {
			break
		}
	}
	return names
}
