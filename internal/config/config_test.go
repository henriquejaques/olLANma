package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestConfigLoadAndSave(t *testing.T) {
	originalHome := os.Getenv("HOME")
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	// 1. Test Load() when file doesn't exist (Should return defaults)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed when no file existed: %v", err)
	}
	defaultCfg := DefaultConfig()
	if cfg != defaultCfg {
		t.Errorf("Load() did not return defaults on missing file. Got: %+v, Want: %+v", cfg, defaultCfg)
	}

	// 2. Test Save() with a valid config
	customCfg := Config{
		Host:         "192.168.1.100",
		Port:         "8080",
		DefaultModel: "gemma3",
		Scheme:       "http",
		Timeout:      120,
	}

	err = Save(customCfg)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// 3. Verify the file actually exists on disk where we expect
	expectedPath := filepath.Join(tempHome, ".config", "ollanma", "config.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("Save() did not write file to the expected path: %s", expectedPath)
	}

	// 4. Test Load() when file exists (Should read the custom config we just saved)
	loadedCfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed reading existing file: %v", err)
	}
	if loadedCfg != customCfg {
		t.Errorf("Load() did not match saved config. Got: %+v, Want: %+v", loadedCfg, customCfg)
	}

	// 5. Test File Permissions are locked down (security requirement)
	info, err := os.Stat(expectedPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("Config file permissions are wrong. Expected -rw------- (0600), got %o", info.Mode().Perm())
	}
}

func TestConfigDirPermissions(t *testing.T) {
	originalHome := os.Getenv("HOME")
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	cfg := DefaultConfig()
	err := Save(cfg)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	configDir := filepath.Join(tempHome, ".config", "ollanma")
	dirInfo, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("Failed to stat config directory: %v", err)
	}
	if dirInfo.Mode().Perm() != 0700 {
		t.Errorf("Config directory permissions are wrong. Expected drwx------ (0700), got %o", dirInfo.Mode().Perm())
	}
}

func TestConfigExists(t *testing.T) {
	originalHome := os.Getenv("HOME")
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	// Should be false before save
	if ConfigExists() {
		t.Error("ConfigExists() returned true before any config was saved")
	}

	// Save a config
	err := Save(DefaultConfig())
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Should be true after save
	if !ConfigExists() {
		t.Error("ConfigExists() returned false after config was saved")
	}
}

func TestValidateHost(t *testing.T) {
	tests := []struct {
		host    string
		wantErr bool
	}{
		{"127.0.0.1", false},
		{"192.168.1.100", false},
		{"10.0.0.1", false},
		{"localhost", false},
		{"my-server.local", true},
		{"", true},
		// H-001: URL-special characters must be rejected
		{"127.0.0.1@evil.com", true},
		{"host/path", true},
		{"host?query", true},
		{"host#fragment", true},
		{"host%00", true},
		{"host\\bad", true},
		// Embedded port should be rejected
		{"192.168.1.1:8080", true},
	}
	for _, tc := range tests {
		err := ValidateHost(tc.host)
		if (err != nil) != tc.wantErr {
			t.Errorf("ValidateHost(%q) error = %v, wantErr = %v", tc.host, err, tc.wantErr)
		}
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		port    string
		wantErr bool
	}{
		{"11434", false},
		{"1", false},
		{"65535", false},
		{"0", true},
		{"65536", true},
		{"-1", true},
		{"abc", true},
		{"", true},
	}
	for _, tc := range tests {
		err := ValidatePort(tc.port)
		if (err != nil) != tc.wantErr {
			t.Errorf("ValidatePort(%q) error = %v, wantErr = %v", tc.port, err, tc.wantErr)
		}
	}
}

func TestValidateScheme(t *testing.T) {
	tests := []struct {
		scheme  string
		wantErr bool
	}{
		{"http", false},
		{"https", false},
		{"ftp", true},
		{"", true},
	}
	for _, tc := range tests {
		err := ValidateScheme(tc.scheme)
		if (err != nil) != tc.wantErr {
			t.Errorf("ValidateScheme(%q) error = %v, wantErr = %v", tc.scheme, err, tc.wantErr)
		}
	}
}

func TestValidateTimeout(t *testing.T) {
	tests := []struct {
		timeout int
		wantErr bool
	}{
		{1, false},
		{300, false},
		{3600, false},
		{0, true},
		{-1, true},
		{3601, true},
		{9999, true},
	}
	for _, tc := range tests {
		err := ValidateTimeout(tc.timeout)
		if (err != nil) != tc.wantErr {
			t.Errorf("ValidateTimeout(%d) error = %v, wantErr = %v", tc.timeout, err, tc.wantErr)
		}
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		host     string
		wantPriv bool
	}{
		{"127.0.0.1", true},
		{"192.168.1.1", true},
		{"10.0.0.5", true},
		{"172.16.0.1", true},
		{"8.8.8.8", false},        // Google DNS — public
		{"169.254.169.254", true}, // Link-local (cloud metadata)
		{"my-server.local", false},
		{"localhost", true},
	}
	for _, tc := range tests {
		got := IsPrivateIP(tc.host)
		if got != tc.wantPriv {
			t.Errorf("IsPrivateIP(%q) = %v, want %v", tc.host, got, tc.wantPriv)
		}
	}
}

func TestValidateConfig_Valid(t *testing.T) {
	// Happy path: a fully valid config should pass
	cfg := Config{
		Host:         "192.168.1.50",
		Port:         "11434",
		DefaultModel: "llama3",
		Scheme:       "https",
		Timeout:      600,
	}
	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("ValidateConfig() rejected a valid config: %v", err)
	}
}

func TestValidateConfig_DefaultsAreValid(t *testing.T) {
	err := ValidateConfig(DefaultConfig())
	if err != nil {
		t.Errorf("ValidateConfig() rejected DefaultConfig(): %v", err)
	}
}

func TestValidateConfig_RejectsHostname(t *testing.T) {
	cfg := Config{
		Host:         "my-server.local",
		Port:         "11434",
		DefaultModel: "llama3",
		Scheme:       "http",
		Timeout:      300,
	}
	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("ValidateConfig() should reject non-localhost hostnames")
	}
}

func TestSaveRejectsInvalidConfig(t *testing.T) {
	originalHome := os.Getenv("HOME")
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	tests := []struct {
		name string
		cfg  Config
	}{
		{"invalid port", Config{Host: "127.0.0.1", Port: "abc", DefaultModel: "llama3", Scheme: "http", Timeout: 300}},
		{"empty model", Config{Host: "127.0.0.1", Port: "11434", DefaultModel: "", Scheme: "http", Timeout: 300}},
		{"invalid scheme", Config{Host: "127.0.0.1", Port: "11434", DefaultModel: "llama3", Scheme: "ftp", Timeout: 300}},
		{"public IP", Config{Host: "8.8.8.8", Port: "11434", DefaultModel: "llama3", Scheme: "http", Timeout: 300}},
		{"invalid timeout", Config{Host: "127.0.0.1", Port: "11434", DefaultModel: "llama3", Scheme: "http", Timeout: 9999}},
		{"empty host", Config{Host: "", Port: "11434", DefaultModel: "llama3", Scheme: "http", Timeout: 300}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := Save(tc.cfg)
			if err == nil {
				t.Errorf("Save() should have rejected config %+v", tc.cfg)
			}
		})
	}
}

func TestLoadRejectsSymlinkedConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior differs on windows and may require elevated privileges")
	}

	originalHome := os.Getenv("HOME")
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	configDir := filepath.Join(tempHome, ".config", "ollanma")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	target := filepath.Join(tempHome, "real-config.json")
	if err := os.WriteFile(target, []byte(`{"host":"127.0.0.1","port":"11434","default_model":"llama3","scheme":"http","timeout":300}`), 0600); err != nil {
		t.Fatalf("failed to write target file: %v", err)
	}

	linkPath := filepath.Join(configDir, "config.json")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Skipf("symlink not available in this environment: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected Load() to reject symlinked config file")
	}
	if !strings.Contains(err.Error(), "symlink is not allowed") {
		t.Fatalf("expected symlink rejection error, got: %v", err)
	}
}

func TestSaveRejectsSymlinkedConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior differs on windows and may require elevated privileges")
	}

	originalHome := os.Getenv("HOME")
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	configDir := filepath.Join(tempHome, ".config", "ollanma")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	target := filepath.Join(tempHome, "real-config.json")
	if err := os.WriteFile(target, []byte("{}"), 0600); err != nil {
		t.Fatalf("failed to write target file: %v", err)
	}

	linkPath := filepath.Join(configDir, "config.json")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Skipf("symlink not available in this environment: %v", err)
	}

	err := Save(DefaultConfig())
	if err == nil {
		t.Fatal("expected Save() to reject symlinked config file")
	}
	if !strings.Contains(err.Error(), "symlink is not allowed") {
		t.Fatalf("expected symlink rejection error, got: %v", err)
	}
}
