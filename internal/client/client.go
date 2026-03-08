package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/henriquejaques/olLANma/internal/config"
)

// allowedCommands is the whitelist of Ollama API endpoints that Forward() will accept.
var allowedCommands = map[string]bool{
	"run":      true,
	"generate": true,
	"chat":     true,
	"pull":     true,
	"rm":       true,
	"cp":       true,
	"list":     true,
	"tags":     true,
	"ps":       true,
	"show":     true,
}

// maxResponseSize is the maximum response body size (100 MB).
// Prevents a malicious server from streaming infinite data.
const maxResponseSize = 100 * 1024 * 1024 // 100 MB

// Forward acts as a simple pass-through client to the remote Ollama server.
// It accepts a context for cancellation support (e.g., SIGINT).
func Forward(ctx context.Context, cfg config.Config, cmd string, args []string) error {
	// Defense-in-depth: validate command against whitelist
	if !allowedCommands[cmd] {
		return fmt.Errorf("unknown command %q — supported commands: run, generate, chat, pull, rm, cp, list, tags, ps, show", cmd)
	}

	if err := config.ValidateHost(cfg.Host); err != nil {
		return fmt.Errorf("invalid host: %w", err)
	}
	if err := config.ValidatePort(cfg.Port); err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	scheme := "http"
	if cfg.Scheme != "" {
		scheme = cfg.Scheme
	}
	if err := config.ValidateScheme(scheme); err != nil {
		return err
	}

	apiEndpoint := cmd
	httpMethod := "POST"
	var payload interface{}

	switch cmd {
	case "run", "generate":
		if len(args) < 2 {
			return fmt.Errorf("%s requires a model name and a prompt", cmd)
		}
		apiEndpoint = "generate"
		model := args[0]
		prompt := strings.Join(args[1:], " ")
		payload = map[string]interface{}{
			"model":  model,
			"prompt": prompt,
		}
	case "chat":
		if len(args) < 2 {
			return fmt.Errorf("chat requires a model name and a prompt")
		}
		model := args[0]
		prompt := strings.Join(args[1:], " ")
		payload = map[string]interface{}{
			"model": model,
			"messages": []map[string]string{
				{
					"role":    "user",
					"content": prompt,
				},
			},
		}
	case "pull", "show":
		if len(args) == 0 {
			return fmt.Errorf("%s requires a model name", cmd)
		}
		if len(args) > 1 {
			return fmt.Errorf("%s received unexpected extra arguments: %v", cmd, args[1:])
		}
		payload = map[string]string{"name": args[0]}
	case "rm", "delete":
		apiEndpoint = "delete"
		httpMethod = "DELETE"
		if len(args) == 0 {
			return fmt.Errorf("rm requires a model name")
		}
		if len(args) > 1 {
			return fmt.Errorf("rm received unexpected extra arguments: %v", args[1:])
		}
		payload = map[string]string{"model": args[0]}
	case "cp":
		apiEndpoint = "copy"
		if len(args) < 2 {
			return fmt.Errorf("cp requires source and destination model names")
		}
		if len(args) > 2 {
			return fmt.Errorf("cp received unexpected extra arguments: %v", args[2:])
		}
		payload = map[string]string{"source": args[0], "destination": args[1]}
	case "list", "tags":
		apiEndpoint = "tags"
		httpMethod = "GET"
		if len(args) > 0 {
			return fmt.Errorf("%s does not accept arguments: %v", cmd, args)
		}
	case "ps":
		httpMethod = "GET"
		if len(args) > 0 {
			return fmt.Errorf("ps does not accept arguments: %v", args)
		}
	}

	baseURL := (&url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(cfg.Host, cfg.Port),
		Path:   "/api/" + apiEndpoint,
	}).String()

	var reqBody []byte
	if payload != nil {
		var err error
		reqBody, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal request payload: %w", err)
		}
	}

	var reqReader io.Reader
	if reqBody != nil {
		reqReader = bytes.NewBuffer(reqBody)
	}

	req, err := http.NewRequestWithContext(ctx, httpMethod, baseURL, reqReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Use the configured timeout (defaults to 300s / 5 min if not set)
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	client := &http.Client{
		Timeout: timeout,
		// Security (M-002): Disable redirects entirely.
		// A compromised server must not redirect requests to external hosts.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach Ollama node — %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("remote API returned status: %s", resp.Status)
	}

	// Stream response back to user's stdout with a size cap (defense against infinite streaming)
	limitedReader := io.LimitReader(resp.Body, maxResponseSize)
	if apiEndpoint == "generate" || apiEndpoint == "chat" {
		decoder := json.NewDecoder(limitedReader)
		isThinking := false
		for {
			var chunk map[string]interface{}
			if err := decoder.Decode(&chunk); err != nil {
				if err == io.EOF {
					if isThinking {
						fmt.Print("\n--- End Thinking ---\n\n")
					}
					break
				}
				// fallback to erroring out
				return fmt.Errorf("error decoding response: %w", err)
			}

			if apiErr, ok := chunk["error"].(string); ok && apiErr != "" {
				return fmt.Errorf("API error: %s", apiErr)
			}

			if thinking, ok := chunk["thinking"].(string); ok && thinking != "" {
				if cfg.ShowThinking {
					if !isThinking {
						fmt.Print("\n--- Thinking ---\n")
						isThinking = true
					}
					fmt.Print(thinking)
				}
			}

			// Transition out of thinking mode when response/content begins
			if isThinking {
				hasResponse := false
				if response, ok := chunk["response"].(string); ok && response != "" {
					hasResponse = true
				}
				if message, ok := chunk["message"].(map[string]interface{}); ok {
					if content, ok := message["content"].(string); ok && content != "" {
						hasResponse = true
					}
				}
				if hasResponse {
					if cfg.ShowThinking {
						fmt.Print("\n--- Response ---\n")
					}
					isThinking = false
				}
			}

			if response, ok := chunk["response"].(string); ok && response != "" {
				fmt.Print(response)
			}

			if message, ok := chunk["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok && content != "" {
					fmt.Print(content)
				}
			}
		}
		if isThinking {
			fmt.Print("\n--- End Thinking ---\n\n")
		}
		fmt.Println()
	} else {
		_, err = io.Copy(os.Stdout, limitedReader)
		if err != nil {
			return fmt.Errorf("error streaming response: %w", err)
		}
	}

	return nil
}
