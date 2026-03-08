package client

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// captureStdout runs fn while capturing os.Stdout and returns the output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestForward_ShowThinking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"thinking":"Hmm...","response":"","done":false}` + "\n"))
		w.Write([]byte(`{"thinking":"","response":"Hello","done":false}` + "\n"))
		w.Write([]byte(`{"thinking":"","response":" World","done":true}` + "\n"))
	}))
	defer server.Close()

	cfg := testCfg(server)
	cfg.ShowThinking = true

	output := captureStdout(t, func() {
		err := Forward(context.Background(), cfg, "generate", []string{"llama3", "hello"})
		if err != nil {
			t.Fatalf("Forward() with ShowThinking=true returned error: %v", err)
		}
	})

	if !strings.Contains(output, "--- Thinking ---") {
		t.Errorf("Expected output to contain '--- Thinking ---', got: %q", output)
	}
	if !strings.Contains(output, "Hmm...") {
		t.Errorf("Expected output to contain 'Hmm...', got: %q", output)
	}
	if !strings.Contains(output, "--- Response ---") {
		t.Errorf("Expected output to contain '--- Response ---', got: %q", output)
	}
	if !strings.Contains(output, "Hello World") {
		t.Errorf("Expected output to contain 'Hello World', got: %q", output)
	}
}

func TestForward_HideThinking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"thinking":"Hmm...","response":"","done":false}` + "\n"))
		w.Write([]byte(`{"thinking":"","response":"Hello","done":false}` + "\n"))
		w.Write([]byte(`{"thinking":"","response":" World","done":true}` + "\n"))
	}))
	defer server.Close()

	cfg := testCfg(server)
	cfg.ShowThinking = false

	output := captureStdout(t, func() {
		err := Forward(context.Background(), cfg, "generate", []string{"llama3", "hello"})
		if err != nil {
			t.Fatalf("Forward() with ShowThinking=false returned error: %v", err)
		}
	})

	if strings.Contains(output, "--- Thinking ---") {
		t.Errorf("Expected output NOT to contain '--- Thinking ---', got: %q", output)
	}
	if strings.Contains(output, "Hmm...") {
		t.Errorf("Expected output NOT to contain 'Hmm...', got: %q", output)
	}
	if strings.Contains(output, "--- Response ---") {
		t.Errorf("Expected output NOT to contain '--- Response ---', got: %q", output)
	}
	if !strings.Contains(output, "Hello World") {
		t.Errorf("Expected output to contain 'Hello World', got: %q", output)
	}
}

func TestForward_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"error":"model 'nonexistent' not found"}` + "\n"))
	}))
	defer server.Close()

	cfg := testCfg(server)
	err := Forward(context.Background(), cfg, "run", []string{"nonexistent", "hello"})
	if err == nil {
		t.Fatal("Expected error for API error response, got nil")
	}
	if !strings.Contains(err.Error(), "API error") {
		t.Errorf("Expected 'API error' in error message, got: %v", err)
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' in error message, got: %v", err)
	}
}

func TestForward_ChatEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("Expected path /api/chat, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":{"role":"assistant","content":"Hi"},"done":false}` + "\n"))
		w.Write([]byte(`{"message":{"role":"assistant","content":" there!"},"done":true}` + "\n"))
	}))
	defer server.Close()

	cfg := testCfg(server)

	output := captureStdout(t, func() {
		err := Forward(context.Background(), cfg, "chat", []string{"llama3", "hello"})
		if err != nil {
			t.Fatalf("Forward() for chat returned error: %v", err)
		}
	})

	if !strings.Contains(output, "Hi there!") {
		t.Errorf("Expected chat output to contain 'Hi there!', got: %q", output)
	}
}

func TestForward_EOFWhileThinking(t *testing.T) {
	// Simulate a stream that ends while still in thinking mode (e.g., connection drop)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"thinking":"Still thinking...","response":"","done":false}` + "\n"))
		// Stream ends without a response chunk
	}))
	defer server.Close()

	cfg := testCfg(server)
	cfg.ShowThinking = true

	output := captureStdout(t, func() {
		err := Forward(context.Background(), cfg, "generate", []string{"llama3", "hello"})
		if err != nil {
			t.Fatalf("Forward() returned error: %v", err)
		}
	})

	if !strings.Contains(output, "--- Thinking ---") {
		t.Errorf("Expected '--- Thinking ---' header, got: %q", output)
	}
	if !strings.Contains(output, "--- End Thinking ---") {
		t.Errorf("Expected '--- End Thinking ---' footer for unclosed thinking block, got: %q", output)
	}
}

func TestForward_NonGenerateEndpointRawOutput(t *testing.T) {
	// Non-generate/chat endpoints (e.g., list/tags) should pass through raw output
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"models":[{"name":"llama3"}]}`))
	}))
	defer server.Close()

	cfg := testCfg(server)

	output := captureStdout(t, func() {
		err := Forward(context.Background(), cfg, "list", []string{})
		if err != nil {
			t.Fatalf("Forward() for list returned error: %v", err)
		}
	})

	if !strings.Contains(output, "llama3") {
		t.Errorf("Expected raw JSON output containing 'llama3', got: %q", output)
	}
}

func TestForward_PullRequiresModelName(t *testing.T) {
	cfg := testCfg(httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))

	err := Forward(context.Background(), cfg, "pull", []string{})
	if err == nil {
		t.Fatal("Expected error for pull without model name, got nil")
	}
	if !strings.Contains(err.Error(), "requires a model name") {
		t.Errorf("Expected 'requires a model name' error, got: %v", err)
	}
}

func TestForward_ShowRequiresModelName(t *testing.T) {
	cfg := testCfg(httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))

	err := Forward(context.Background(), cfg, "show", []string{})
	if err == nil {
		t.Fatal("Expected error for show without model name, got nil")
	}
	if !strings.Contains(err.Error(), "requires a model name") {
		t.Errorf("Expected 'requires a model name' error, got: %v", err)
	}
}

func TestForward_RmRequiresModelName(t *testing.T) {
	cfg := testCfg(httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))

	err := Forward(context.Background(), cfg, "rm", []string{})
	if err == nil {
		t.Fatal("Expected error for rm without model name, got nil")
	}
	if !strings.Contains(err.Error(), "requires a model name") {
		t.Errorf("Expected 'requires a model name' error, got: %v", err)
	}
}

func TestForward_CpRequiresTwoArgs(t *testing.T) {
	cfg := testCfg(httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))

	err := Forward(context.Background(), cfg, "cp", []string{"only-one"})
	if err == nil {
		t.Fatal("Expected error for cp with only one arg, got nil")
	}
	if !strings.Contains(err.Error(), "requires source and destination") {
		t.Errorf("Expected 'requires source and destination' error, got: %v", err)
	}
}

func TestForward_DeleteCorrectMethod(t *testing.T) {
	var receivedMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := testCfg(server)
	err := Forward(context.Background(), cfg, "rm", []string{"old-model"})
	if err != nil {
		t.Fatalf("Forward() for rm returned error: %v", err)
	}
	if receivedMethod != "DELETE" {
		t.Errorf("Expected DELETE method for rm, got %s", receivedMethod)
	}
}

func TestForward_CpCorrectPayload(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := testCfg(server)
	err := Forward(context.Background(), cfg, "cp", []string{"model-a", "model-b"})
	if err != nil {
		t.Fatalf("Forward() for cp returned error: %v", err)
	}
	if receivedPath != "/api/copy" {
		t.Errorf("Expected path /api/copy, got %s", receivedPath)
	}
}

func TestForward_PsUsesGET(t *testing.T) {
	var receivedMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := testCfg(server)
	err := Forward(context.Background(), cfg, "ps", []string{})
	if err != nil {
		t.Fatalf("Forward() for ps returned error: %v", err)
	}
	if receivedMethod != "GET" {
		t.Errorf("Expected GET method for ps, got %s", receivedMethod)
	}
}

func TestForward_DefaultSchemeIsHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := testCfg(server)
	cfg.Scheme = "" // empty scheme should default to http

	err := Forward(context.Background(), cfg, "list", []string{})
	if err != nil {
		t.Fatalf("Forward() with empty scheme returned error: %v", err)
	}
}

func TestForward_GenerateEndpointPath(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"response":"ok","done":true}` + "\n"))
	}))
	defer server.Close()

	cfg := testCfg(server)
	err := Forward(context.Background(), cfg, "generate", []string{"llama3", "hello"})
	if err != nil {
		t.Fatalf("Forward() returned error: %v", err)
	}
	if receivedPath != "/api/generate" {
		t.Errorf("Expected path /api/generate, got %s", receivedPath)
	}
}
