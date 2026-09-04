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

	})
}

func TestConfiguredRun(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "inject")
	build := exec.Command("go", "build", "-o", binaryPath, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}

	projectDir := t.TempDir()
	config := `format_version = 1
project_id = "test-project"

[profiles.default]
provider = "1password"
account = "acme"
vault = "Engineering"
item_id = "stable-note-id"

[commands.show-token]
profile = "default"
command = ["sh", "-c", "printf %s \"$TOKEN\""]
`
	if err := os.WriteFile(filepath.Join(projectDir, "inject.toml"), []byte(config), 0o600); err != nil {
		t.Fatalf("write inject.toml: %v", err)
	}
	opPath := filepath.Join(projectDir, "op")
	op := "#!/bin/sh\nprintf '%s\\n' '{\"fields\":[{\"id\":\"notesPlain\",\"value\":\"TOKEN=from-configured-note\\n\"}]}'\n"
	if err := os.WriteFile(opPath, []byte(op), 0o700); err != nil {
		t.Fatalf("write fake op: %v", err)
	}

	command := exec.Command(binaryPath, "run", "--", "sh", "-c", `printf %s "$TOKEN"`)
	command.Dir = projectDir
	command.Env = append(os.Environ(), "PATH="+projectDir+string(os.PathListSeparator)+os.Getenv("PATH"), "TOKEN=parent-value")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run configured command: %v\n%s", err, output)
	}
	if got := string(output); got != "from-configured-note" {
		t.Errorf("injected output = %q, want configured secret", got)
	}

	binding := exec.Command(binaryPath, "show-token")
	binding.Dir = projectDir
	binding.Env = append(os.Environ(), "PATH="+projectDir+string(os.PathListSeparator)+os.Getenv("PATH"), "TOKEN=parent-value")
	output, err = binding.CombinedOutput()
	if err != nil {
		t.Fatalf("run configured binding: %v\n%s", err, output)
	}
	if got := string(output); got != "from-configured-note" {
		t.Errorf("bound command output = %q, want configured secret", got)
	}
}

func TestConfiguredRunDoesNotLaunchChildWhenSourceFails(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "inject")
	build := exec.Command("go", "build", "-o", binaryPath, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}

	projectDir := t.TempDir()
	config := `format_version = 1
project_id = "test-project"
[profiles.default]
provider = "1password"
account = "acme"
vault = "Engineering"
item_id = "stable-note-id"
`
	if err := os.WriteFile(filepath.Join(projectDir, "inject.toml"), []byte(config), 0o600); err != nil {
		t.Fatalf("write inject.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "op"), []byte("#!/bin/sh\nexit 1\n"), 0o700); err != nil {
		t.Fatalf("write failing fake op: %v", err)
	}
	sentinel := filepath.Join(projectDir, "child-was-launched")

	command := exec.Command(binaryPath, "run", "--", "sh", "-c", "touch \"$1\"", "sh", sentinel)
	command.Dir = projectDir
	command.Env = append(os.Environ(), "PATH="+projectDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if output, err := command.CombinedOutput(); err == nil {
		t.Fatalf("run configured command = success, want source failure; output: %s", output)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Errorf("child launch sentinel stat error = %v, want not exist", err)
	}
}

func TestSetupWritesConfirmedConfigurationAfterValidation(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "inject")
	build := exec.Command("go", "build", "-o", binaryPath, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}

	projectDir := t.TempDir()
	opPath := filepath.Join(projectDir, "op")
	op := "#!/bin/sh\nif [ \"$1\" = account ]; then exit 0; fi\nprintf '%s\\n' '{\"fields\":[{\"id\":\"notesPlain\",\"value\":\"TOKEN=from-note\\n\"}]}'\n"
	if err := os.WriteFile(opPath, []byte(op), 0o700); err != nil {
		t.Fatalf("write fake op: %v", err)
	}

	command := exec.Command(binaryPath,
		"setup", "--project-id", "billing-api", "--account", "acme", "--vault", "Engineering", "--item-id", "stable-note-id", "--yes",
		"--validate=sh", "--validate=-c", "--validate=exit 0",
	)
	command.Dir = projectDir
	command.Env = append(os.Environ(), "PATH="+projectDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run setup: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "Validation succeeded") {
		t.Errorf("setup output = %q, want validation confirmation", output)
	}
	config, err := os.ReadFile(filepath.Join(projectDir, "inject.toml"))
	if err != nil {
		t.Fatalf("read inject.toml: %v", err)
	}
	if strings.Contains(string(config), "TOKEN=") || !strings.Contains(string(config), "item_id = \"stable-note-id\"") {
		t.Errorf("inject.toml = %q, want non-secret remote reference", config)
	}
}
