package config

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config represents the user's settings
type Config struct {
	Host         string `json:"host"`
	Port         string `json:"port"`
	DefaultModel string `json:"default_model"`
	Scheme       string `json:"scheme,omitempty"`  // "http" or "https", defaults to "http"
	Timeout      int    `json:"timeout,omitempty"` // HTTP timeout in seconds, defaults to 300 (5 min)
	ShowThinking bool   `json:"show_thinking"`
}

const (
	configDirName  = "ollanma"
	configFileName = "config.json"
)

// DefaultConfig returns the default settings
func DefaultConfig() Config {
	return Config{
		Host:         "127.0.0.1",
		Port:         "11434",
		DefaultModel: "llama3",
		Scheme:       "http",
		Timeout:      300,
		ShowThinking: true,
	}
}

// GetConfigPath computes the path: ~/.config/ollanma/config.json
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// Use standard XDG base dir concept as fallback for cross-platform (mostly Linux target)
	configDir := filepath.Join(homeDir, ".config", configDirName)
	return filepath.Join(configDir, configFileName), nil
}

// strictConfigPath returns the canonical expected config path and rejects
// unexpected drift (defense-in-depth against path confusion).
func strictConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	expected := filepath.Clean(filepath.Join(homeDir, ".config", configDirName, configFileName))

	path, err := GetConfigPath()
	if err != nil {
		return "", err
	}
	clean := filepath.Clean(path)
	if clean != expected {
		return "", fmt.Errorf("unexpected config path %q (expected %q)", clean, expected)
	}
	return clean, nil
}

func ensureNotSymlink(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlink is not allowed for security-sensitive path: %s", path)
	}
	return nil
}

// ConfigExists returns true if the config file already exists on disk.
func ConfigExists() bool {
	path, err := GetConfigPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// ValidateHost checks that the host is a safe LAN target.
// Only localhost and numeric IPv4 addresses are accepted.
func ValidateHost(host string) error {
	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	// Reject URL-special characters that could cause misrouting (H-001)
	if strings.ContainsAny(host, "@/?#%\\") {
		return fmt.Errorf("host contains invalid characters (@ / ? # %% \\): %q", host)
	}
	// Reject hosts with embedded port (colons are only valid inside IPv6 brackets)
	if strings.Contains(host, ":") && !strings.Contains(host, "[") {
		return fmt.Errorf("host must not contain a port — use the port field instead: %q", host)
	}
	// Accept "localhost"
	if host == "localhost" {
		return nil
	}
	// Accept numeric IPv4 only. This avoids DNS-based bypasses for LAN-only policy.
	ip := net.ParseIP(host)
	if ip == nil || ip.To4() == nil {
		return fmt.Errorf("host must be 'localhost' or an IPv4 address, got %q", host)
	}
	return nil
}

// ValidatePort checks that the port is a valid number in range 1-65535.
func ValidatePort(port string) error {
	p, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("port must be a number, got %q", port)
	}
	if p < 1 || p > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", p)
	}
	return nil
}

// ValidateScheme checks that the scheme is either "http" or "https".
func ValidateScheme(scheme string) error {
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("scheme must be 'http' or 'https', got %q", scheme)
	}
	return nil
}

// ValidateTimeout checks that the timeout is a positive number of seconds.
func ValidateTimeout(timeout int) error {
	if timeout < 1 {
		return fmt.Errorf("timeout must be at least 1 second, got %d", timeout)
	}
	if timeout > 3600 {
		return fmt.Errorf("timeout must be at most 3600 seconds (1 hour), got %d", timeout)
	}
	return nil
}

// IsPrivateIP checks whether a host is localhost or an IPv4 address in loopback/private/link-local ranges.
func IsPrivateIP(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil || ip.To4() == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

// ValidateConfig runs all field validations on a Config.
func ValidateConfig(cfg Config) error {
	if err := ValidateHost(cfg.Host); err != nil {
		return fmt.Errorf("invalid host: %w", err)
	}
	if err := ValidatePort(cfg.Port); err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}
	if cfg.DefaultModel == "" {
		return fmt.Errorf("default model cannot be empty")
	}
	if cfg.Scheme != "" {
		if err := ValidateScheme(cfg.Scheme); err != nil {
			return err
		}
	}
	if cfg.Timeout != 0 {
		if err := ValidateTimeout(cfg.Timeout); err != nil {
			return err
		}
	}
	if !IsPrivateIP(cfg.Host) {
		return fmt.Errorf("host %q is not a private/LAN target. Use localhost or a private IPv4 address (10.x.x.x, 172.16-31.x.x, 192.168.x.x, 127.x.x.x)", cfg.Host)
	}
	return nil
}

// Load reads the config from disk or returns defaults if it doesn't exist.
// It checks file permissions and re-validates the loaded config (M-001).
func Load() (Config, error) {
	cfg := DefaultConfig()

	path, err := strictConfigPath()
	if err != nil {
		return cfg, err // graceful failure falls back to defaults
	}
	configDir := filepath.Dir(path)

	// Reject symlinked config files/directories to prevent path redirection.
	if err := ensureNotSymlink(configDir); err != nil {
		if !os.IsNotExist(err) {
			return cfg, err
		}
		return cfg, nil // Not an error if it just hasn't been created yet
	}

	root, err := os.OpenRoot(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	defer root.Close()

	info, err := root.Lstat(configFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return cfg, fmt.Errorf("symlink is not allowed for security-sensitive path: %s", path)
	}
	if !info.Mode().IsRegular() {
		return cfg, fmt.Errorf("config path is not a regular file: %s", path)
	}
	data, err := root.ReadFile(configFileName)
	if err != nil {
		return cfg, err
	}

	// Security: check file permissions are not too open
	checkPermissions(path)

	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Re-validate loaded config to catch manual edits or corruption (M-001)
	if err := ValidateConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: loaded config is invalid (%v), falling back to defaults\n", err)
		return DefaultConfig(), nil
	}

	return cfg, nil
}

// checkPermissions warns if the config file or its parent directory have
// overly permissive permissions (writable/readable by group or others).
func checkPermissions(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	perm := info.Mode().Perm()
	if perm&0077 != 0 {
		fmt.Fprintf(os.Stderr, "Warning: config file %s has permissions %o (expected 0600). "+
			"Run: chmod 600 %s\n", path, perm, path)
	}

	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		return
	}
	dirPerm := dirInfo.Mode().Perm()
	if dirPerm&0077 != 0 {
		fmt.Fprintf(os.Stderr, "Warning: config directory has permissions %o (expected 0700). "+
			"Run: chmod 700 %s\n", dirPerm, filepath.Dir(path))
	}
}

// Save writes the given config to disk after validation
func Save(cfg Config) error {
	if err := ValidateConfig(cfg); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	path, err := strictConfigPath()
	if err != nil {
		return err
	}

	configDir := filepath.Dir(path)

	// Existing symlinked directories/files are rejected to avoid redirection.
	if err := ensureNotSymlink(configDir); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}
	if err := ensureNotSymlink(path); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	// Ensure the parent directory exists with secure permissions
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}
	if err := ensureNotSymlink(configDir); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}

	// Atomic write with restricted permissions.
	tmp, err := os.CreateTemp(configDir, "config.json.tmp-*")
	if err != nil {
		return fmt.Errorf("could not create temp config file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err := tmp.Chmod(0600); err != nil {
		closeErr := tmp.Close()
		if closeErr != nil {
			return fmt.Errorf("could not set temp config permissions: %w (and close failed: %v)", err, closeErr)
		}
		return fmt.Errorf("could not set temp config permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		closeErr := tmp.Close()
		if closeErr != nil {
			return fmt.Errorf("could not write temp config: %w (and close failed: %v)", err, closeErr)
		}
		return fmt.Errorf("could not write temp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("could not close temp config: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("could not atomically replace config: %w", err)
	}
	return nil
}
