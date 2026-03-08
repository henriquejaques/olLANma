package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func captureStdoutMain(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()
	_ = w.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to capture stdout: %v", err)
	}
	_ = r.Close()
	return buf.String()
}

func TestPrintUsage_IncludesSupportedCommands(t *testing.T) {
	out := captureStdoutMain(t, printUsage)

	expected := []string{
		"olLANma — Remote CLI client for Ollama on your LAN",
		"Supported passthrough commands:",
		"run, generate, chat, pull, rm, cp, list, tags, ps, show",
		"olLANma chat <model> <prompt>",
		"olLANma generate <model> <prompt>",
	}

	for _, token := range expected {
		if !strings.Contains(out, token) {
			t.Fatalf("printUsage() missing expected token %q\nOutput:\n%s", token, out)
		}
	}

	if strings.Contains(out, "olLANma serve") || strings.Contains(out, "olLANma create") {
		t.Fatalf("printUsage() should not advertise unsupported commands\nOutput:\n%s", out)
	}
}

func TestMain_HelpFlagExitsZero(t *testing.T) {
	out, code := runMainInSubprocess(t, "--help")
	if code != 0 {
		t.Fatalf("expected exit code 0 for --help, got %d\nOutput:\n%s", code, out)
	}
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("expected usage output for --help, got:\n%s", out)
	}
}

func TestMain_VersionFlagExitsZero(t *testing.T) {
	out, code := runMainInSubprocess(t, "--version")
	if code != 0 {
		t.Fatalf("expected exit code 0 for --version, got %d\nOutput:\n%s", code, out)
	}
	if !strings.Contains(out, "olLANma") {
		t.Fatalf("expected version output, got:\n%s", out)
	}
}

func TestMain_SkipSetupWithoutCommandExitsOne(t *testing.T) {
	out, code := runMainInSubprocess(t, "--skip-setup")
	if code != 1 {
		t.Fatalf("expected exit code 1 for missing command, got %d\nOutput:\n%s", code, out)
	}
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("expected usage output for missing command, got:\n%s", out)
	}
}

func TestHelperMainProcess(t *testing.T) {
	if os.Getenv("GO_WANT_MAIN_HELPER_PROCESS") != "1" {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			os.Exit(2)
		}
	}()

	// Pass args after '--' through to main().
	idx := 0
	for i, arg := range os.Args {
		if arg == "--" {
			idx = i + 1
			break
		}
	}
	progArgs := []string{"olLANma"}
	if idx > 0 && idx <= len(os.Args) {
		progArgs = append(progArgs, os.Args[idx:]...)
	}
	os.Args = progArgs
	main()
}

func runMainInSubprocess(t *testing.T, args ...string) (output string, exitCode int) {
	t.Helper()

	cmdArgs := []string{"-test.run=TestHelperMainProcess", "--"}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command(os.Args[0], cmdArgs...)
	cmd.Env = append(os.Environ(), "GO_WANT_MAIN_HELPER_PROCESS=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		return string(out), 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return string(out), exitErr.ExitCode()
	}
	t.Fatalf("failed to run subprocess: %v", err)
	return "", -1
}
