package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIIdentity(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "inject")
	build := exec.Command("go", "build", "-o", binaryPath, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}

	t.Run("inject is the primary command", func(t *testing.T) {
		output, err := exec.Command(binaryPath, "--help").CombinedOutput()
		if err != nil {
			t.Fatalf("run inject help: %v\n%s", err, output)
		}
		if !strings.Contains(string(output), "Usage:\n  inject [command]") {
			t.Fatalf("inject help = %q, want primary product name", output)
		}
	})

	t.Run("env-pull remains a legacy alias", func(t *testing.T) {
		legacyPath := filepath.Join(t.TempDir(), "env-pull")
		if err := os.Symlink(binaryPath, legacyPath); err != nil {
			t.Fatalf("create legacy alias: %v", err)
		}

		output, err := exec.Command(legacyPath, "--help").CombinedOutput()
		if err != nil {
			t.Fatalf("run env-pull help: %v\n%s", err, output)
		}
		if !strings.Contains(string(output), "env-pull is deprecated; use inject instead") {
			t.Fatalf("legacy invocation = %q, want migration notice", output)
		}

		legacyCommand := exec.Command(legacyPath, "run", "--", "go", "version")
		output, err = legacyCommand.CombinedOutput()
		if err != nil {
			t.Fatalf("run child through env-pull: %v\n%s", err, output)
		}
		if !strings.Contains(string(output), "go version") {
			t.Fatalf("legacy child output = %q, want go version", output)
		}
	})
}
