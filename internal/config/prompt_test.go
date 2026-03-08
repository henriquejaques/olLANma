package config

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func cfgForServer(t *testing.T, server *httptest.Server) Config {
	t.Helper()
	addr, ok := server.Listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("unexpected listener addr type: %T", server.Listener.Addr())
	}
	return Config{
		Host:   addr.IP.String(),
		Port:   fmt.Sprintf("%d", addr.Port),
		Scheme: "http",
	}
}

func TestIsSafeModelName(t *testing.T) {
	tests := []struct {
		name string
		ok   bool
	}{
		{"llama3", true},
		{"gemma2:latest", true},
		{"", false},
		{strings.Repeat("a", maxDiscoveredModelNameLength+1), false},
		{"bad\nname", false},
		{"bad\x1b[31mname", false},
		{"bad\x00name", false},
	}
	for _, tc := range tests {
		if got := isSafeModelName(tc.name); got != tc.ok {
			t.Errorf("isSafeModelName(%q) = %v, want %v", tc.name, got, tc.ok)
		}
	}
}

func TestHostAllowedForModelDiscovery(t *testing.T) {
	tests := []struct {
		host string
		ok   bool
	}{
		{"localhost", true},
		{"127.0.0.1", true},
		{"192.168.1.10", true},
		{"10.0.0.12", true},
		{"8.8.8.8", false},
	}
	for _, tc := range tests {
		if got := hostAllowedForModelDiscovery(tc.host); got != tc.ok {
			t.Errorf("hostAllowedForModelDiscovery(%q) = %v, want %v", tc.host, got, tc.ok)
		}
	}
}

func TestFetchAvailableModels_FiltersUnsafeAndDuplicate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"models":[{"name":"llama3"},{"name":"llama3"},{"name":"bad\nname"},{"name":"  gemma2  "}]}`)
	}))
	defer server.Close()

	cfg := cfgForServer(t, server)
	got := fetchAvailableModels(cfg)

	want := []string{"llama3", "gemma2"}
	if len(got) != len(want) {
		t.Fatalf("fetchAvailableModels() len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("fetchAvailableModels()[%d] = %q, want %q (all=%v)", i, got[i], want[i], got)
		}
	}
}

func TestResolveDefaultModelInput_WithDiscoveredModels(t *testing.T) {
	models := []string{"llama3", "gemma2", "mistral"}

	tests := []struct {
		in      string
		want    string
		wantOK  bool
		wantErr string
	}{
		{"1", "llama3", true, ""},
		{"2", "gemma2", true, ""},
		{"=1", "1", true, ""},
		{"999", "", false, "out of range"},
		{"-1", "", false, "out of range"},
		{"bad\nname", "", false, "control characters"},
		{"custom-model", "custom-model", true, ""},
	}

	for _, tc := range tests {
		got, ok, errMsg := resolveDefaultModelInput(tc.in, models)
		if ok != tc.wantOK {
			t.Fatalf("resolveDefaultModelInput(%q) ok=%v want %v", tc.in, ok, tc.wantOK)
		}
		if got != tc.want {
			t.Fatalf("resolveDefaultModelInput(%q) model=%q want %q", tc.in, got, tc.want)
		}
		if tc.wantErr != "" && !strings.Contains(errMsg, tc.wantErr) {
			t.Fatalf("resolveDefaultModelInput(%q) err=%q expected to contain %q", tc.in, errMsg, tc.wantErr)
		}
	}
}

func TestResolveDefaultModelInput_NoDiscoveredModels(t *testing.T) {
	got, ok, errMsg := resolveDefaultModelInput("  my-model  ", nil)
	if !ok || got != "my-model" {
		t.Fatalf("expected literal model acceptance, got ok=%v model=%q err=%q", ok, got, errMsg)
	}
}
