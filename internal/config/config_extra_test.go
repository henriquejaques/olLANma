package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCorruptJSON(t *testing.T) {
	originalHome := os.Getenv("HOME")
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	configDir := filepath.Join(tempHome, ".config", "ollanma")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	// Write corrupt JSON
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{not valid json!!!`), 0600); err != nil {
		t.Fatalf("failed to write corrupt config: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Expected Load() to return error for corrupt JSON, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("Expected 'failed to parse' error, got: %v", err)
	}
}

func TestLoadInvalidConfigFallsBackToDefaults(t *testing.T) {
	originalHome := os.Getenv("HOME")
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	configDir := filepath.Join(tempHome, ".config", "ollanma")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	// Write valid JSON but with invalid config values (public IP)
	badCfg := Config{
		Host:         "8.8.8.8",
		Port:         "11434",
		DefaultModel: "llama3",
		Scheme:       "http",
		Timeout:      300,
	}
	data, _ := json.MarshalIndent(badCfg, "", "  ")
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() should not return error for invalid config (should fall back), got: %v", err)
	}

	defaults := DefaultConfig()
	if cfg != defaults {
		t.Errorf("Expected Load() to fall back to defaults for invalid config. Got: %+v, Want: %+v", cfg, defaults)
	}
}

func TestLoadShowThinkingPersistence(t *testing.T) {
	originalHome := os.Getenv("HOME")
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	// Save with ShowThinking = false
	cfg := DefaultConfig()
	cfg.ShowThinking = false
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if loaded.ShowThinking != false {
		t.Error("Expected ShowThinking=false after save/load, got true")
	}

	// Save with ShowThinking = true
	cfg.ShowThinking = true
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	loaded, err = Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if loaded.ShowThinking != true {
		t.Error("Expected ShowThinking=true after save/load, got false")
	}
}

func TestCheckPermissionsWarnsOnLoosePerms(t *testing.T) {
	originalHome := os.Getenv("HOME")
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	configDir := filepath.Join(tempHome, ".config", "ollanma")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	checkPermissions(configPath)

	w.Close()
	os.Stderr = oldStderr

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "Warning") {
		t.Errorf("Expected permission warning on stderr, got: %q", output)
	}
}

func TestSaveCreatesDirectoryIfMissing(t *testing.T) {
	originalHome := os.Getenv("HOME")
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	// Ensure directory doesn't exist
	configDir := filepath.Join(tempHome, ".config", "ollanma")
	if _, err := os.Stat(configDir); err == nil {
		t.Fatal("Config directory should not exist yet")
	}

	err := Save(DefaultConfig())
	if err != nil {
		t.Fatalf("Save() failed when directory didn't exist: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Fatal("Save() did not create the config directory")
	}
}

func TestValidateConfig_EmptySchemeAndTimeoutAreOK(t *testing.T) {
	cfg := Config{
		Host:         "192.168.1.50",
		Port:         "11434",
		DefaultModel: "llama3",
		Scheme:       "", // empty scheme should be fine (defaults later)
		Timeout:      0,  // zero timeout should be fine (defaults later)
	}
	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("ValidateConfig() should accept empty scheme and zero timeout, got: %v", err)
	}
}

func TestLoadNotRegularFile(t *testing.T) {
	originalHome := os.Getenv("HOME")
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	configDir := filepath.Join(tempHome, ".config", "ollanma")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	// Create config.json as a directory instead of a file
	configPath := filepath.Join(configDir, "config.json")
	if err := os.MkdirAll(configPath, 0700); err != nil {
		t.Fatalf("failed to create config as directory: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Expected Load() to error when config.json is a directory")
	}
	if !strings.Contains(err.Error(), "not a regular file") {
		t.Errorf("Expected 'not a regular file' error, got: %v", err)
	}
}
