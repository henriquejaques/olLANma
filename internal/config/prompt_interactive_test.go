package config

import (
	"os"
	"testing"
)

func withPromptInput(t *testing.T, input string, fn func()) {
	t.Helper()

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		r.Close()
	}()

	if _, err := w.WriteString(input); err != nil {
		t.Fatalf("failed to write prompt input: %v", err)
	}
	_ = w.Close()

	fn()
}

func TestRunInteractivePrompt_SavesExplicitValues(t *testing.T) {
	originalHome := os.Getenv("HOME")
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	// host, port, scheme, timeout, show_thinking, default_model
	input := "127.0.0.1\n1\nhttp\n5\nn\nllama3\n"

	withPromptInput(t, input, func() {
		if err := RunInteractivePrompt(); err != nil {
			t.Fatalf("RunInteractivePrompt() returned error: %v", err)
		}
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed after prompt save: %v", err)
	}

	if cfg.Host != "127.0.0.1" {
		t.Fatalf("expected host 127.0.0.1, got %q", cfg.Host)
	}
	if cfg.Port != "1" {
		t.Fatalf("expected port 1, got %q", cfg.Port)
	}
	if cfg.Scheme != "http" {
		t.Fatalf("expected scheme http, got %q", cfg.Scheme)
	}
	if cfg.Timeout != 5 {
		t.Fatalf("expected timeout 5, got %d", cfg.Timeout)
	}
	if cfg.ShowThinking {
		t.Fatal("expected ShowThinking=false")
	}
	if cfg.DefaultModel != "llama3" {
		t.Fatalf("expected default model llama3, got %q", cfg.DefaultModel)
	}
}

func TestRunInteractivePrompt_RePromptsOnInvalidInput(t *testing.T) {
	originalHome := os.Getenv("HOME")
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	// invalid host -> valid host
	// invalid port -> valid port
	// invalid scheme -> valid scheme
	// invalid timeout -> valid timeout
	// invalid thinking toggle -> valid toggle
	// valid default model
	input := "example.com\n127.0.0.1\nabc\n1\nftp\nhttp\n0\n10\nmaybe\ny\nllama3\n"

	withPromptInput(t, input, func() {
		if err := RunInteractivePrompt(); err != nil {
			t.Fatalf("RunInteractivePrompt() returned error: %v", err)
		}
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed after prompt save: %v", err)
	}

	if cfg.Host != "127.0.0.1" {
		t.Fatalf("expected host 127.0.0.1 after retry, got %q", cfg.Host)
	}
	if cfg.Port != "1" {
		t.Fatalf("expected port 1 after retry, got %q", cfg.Port)
	}
	if cfg.Scheme != "http" {
		t.Fatalf("expected scheme http after retry, got %q", cfg.Scheme)
	}
	if cfg.Timeout != 10 {
		t.Fatalf("expected timeout 10 after retry, got %d", cfg.Timeout)
	}
	if !cfg.ShowThinking {
		t.Fatal("expected ShowThinking=true after retry")
	}
}
