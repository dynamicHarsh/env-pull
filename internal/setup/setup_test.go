package setup_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/harsh-sonkar/env-pull/internal/setup"
	"github.com/harsh-sonkar/env-pull/internal/store"
)

func TestRunDoesNotChangeProjectWhenOnePasswordIsUnavailable(t *testing.T) {
	directory := t.TempDir()
	envPath := filepath.Join(directory, ".env")
	packagePath := filepath.Join(directory, "package.json")
	envContents := []byte("TOKEN=secret-value\n")
	packageContents := []byte(`{"scripts":{"check":"go test ./..."}}`)
	if err := os.WriteFile(envPath, envContents, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(packagePath, packageContents, 0o600); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	err := setup.Run(setup.Request{
		Directory: directory,
		ProjectID: "billing-api",
		Account:   "acme",
		Vault:     "Engineering",
		ItemID:    "note-id",
		Confirm:   true,
		CheckOnePassword: func() error {
			return setup.ErrOnePasswordUnavailable
		},
		Output: &output,
	})
	if err == nil {
		t.Fatal("Run() error = nil, want unavailable 1Password error")
	}
	if got := output.String(); !strings.Contains(got, "Setup plan:") || !strings.Contains(got, "1Password is unavailable; run `op signin` and retry") || strings.Contains(got, "secret-value") {
		t.Errorf("output = %q, want non-secret plan followed by unavailable provider guidance", got)
	}
	if _, err := os.Stat(filepath.Join(directory, "inject.toml")); !os.IsNotExist(err) {
		t.Errorf("inject.toml stat error = %v, want no configuration written", err)
	}
	if got, err := os.ReadFile(envPath); err != nil || !bytes.Equal(got, envContents) {
		t.Errorf(".env = %q, %v; want unchanged", got, err)
	}
	if got, err := os.ReadFile(packagePath); err != nil || !bytes.Equal(got, packageContents) {
		t.Errorf("package.json = %q, %v; want unchanged", got, err)
	}
}

func TestRunPreviewsConfigurationUntilConfirmed(t *testing.T) {
	directory := t.TempDir()
	var output bytes.Buffer
	err := setup.Run(request(directory, &output))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := output.String(); !bytes.Contains([]byte(got), []byte("Will write inject.toml:")) || bytes.Contains([]byte(got), []byte("secret-value")) {
		t.Errorf("preview = %q, want non-secret configuration preview", got)
	}
	if _, err := os.Stat(filepath.Join(directory, "inject.toml")); !os.IsNotExist(err) {
		t.Errorf("inject.toml stat error = %v, want no configuration without confirmation", err)
	}
}

func TestRunPrefersPackageNameForDetectedProjectID(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, ".env"), []byte("TOKEN=secret-value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "package.json"), []byte(`{"name":"billing-api"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	credentialStore := store.NewMemory()
	var output bytes.Buffer

	if err := setup.Run(setup.Request{Directory: directory, Store: credentialStore, Output: &output}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := output.String(); !strings.Contains(got, `project_id = "billing-api"`) {
		t.Errorf("preview = %q, want package name as detected project ID", got)
	}
	if _, err := os.Stat(filepath.Join(directory, "inject.toml")); !os.IsNotExist(err) {
		t.Errorf("inject.toml stat error = %v, want no configuration without confirmation", err)
	}
	if _, err := credentialStore.Get("billing-api", "default"); err == nil {
		t.Error("Get() error = nil after preview, want unavailable secret set")
	}
}

func TestRunPreviewsCompleteDetectedPlanWithoutMutation(t *testing.T) {
	directory := t.TempDir()
	files := map[string]string{
		".env":                    "BASE_TOKEN=base-secret\n",
		".env.staging":            "API_TOKEN=staging-secret\n",
		".env.production.example": "API_TOKEN=placeholder-secret\n",
		".env.backup":             "API_TOKEN=backup-secret\n",
		"pnpm-lock.yaml":          "lockfileVersion: '9.0'\n",
		"package.json":            `{"name":"billing-api","scripts":{"start":"node .","dev":"vite","serve":"http-server","test":"go test ./...","lint":"eslint .","build":"vite build","release":"shipit"}}`,
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	credentialStore := store.NewMemory()
	var output bytes.Buffer

	if err := setup.Run(setup.Request{Directory: directory, Validate: []string{"pnpm", "test"}, Store: credentialStore, Output: &output}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	preview := output.String()
	for _, want := range []string{
		"Setup plan:",
		"Project ID: billing-api",
		"Source choices: Local (selected), 1Password, Bitwarden",
		"Plaintext input: .env (selected; profile default)",
		"Plaintext input: .env.staging (available; profile staging; precedence .env < .env.staging)",
		"Developer command: dev (default)",
		"Developer command: start (default)",
		"Developer command: serve (default)",
		"Developer command: release",
		"Developer command: build",
		"Developer command: lint",
		"Developer command: test",
		"Validation candidate: build",
		"Validation candidate: lint",
		"Validation candidate: test",
		`Selected validation: ["pnpm","test"]`,
		"Package manager: pnpm",
		"File change: create inject.toml",
	} {
		if !strings.Contains(preview, want) {
			t.Errorf("preview = %q, want %q", preview, want)
		}
	}
	for _, excluded := range []string{".env.production.example", ".env.backup", "base-secret", "staging-secret", "placeholder-secret", "backup-secret"} {
		if strings.Contains(preview, excluded) {
			t.Errorf("preview = %q, must exclude %q", preview, excluded)
		}
	}
	if _, err := os.Stat(filepath.Join(directory, "inject.toml")); !os.IsNotExist(err) {
		t.Errorf("inject.toml stat error = %v, want no configuration without confirmation", err)
	}
	if _, err := credentialStore.Get("billing-api", "default"); err == nil {
		t.Error("Get() error = nil after preview, want unavailable secret set")
	}
}

func TestRunRequiresExplicitSourceWithoutPlaintextInput(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "package.json"), []byte(`{"name":"billing-api"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, ".env.example"), []byte("TOKEN=placeholder-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	credentialStore := store.NewMemory()
	var output bytes.Buffer

	err := setup.Run(setup.Request{Directory: directory, Store: credentialStore, Output: &output})
	if err == nil || err.Error() != "setup: source is required (local, 1password, or bitwarden)" {
		t.Fatalf("Run() error = %v, want explicit source error", err)
	}
	if output.Len() != 0 {
		t.Errorf("output = %q, want no incomplete preview", output.String())
	}
	if _, err := os.Stat(filepath.Join(directory, "inject.toml")); !os.IsNotExist(err) {
		t.Errorf("inject.toml stat error = %v, want no configuration after invalid input", err)
	}
}

func TestRunDetectsAndValidatesStableProjectID(t *testing.T) {
	t.Run("Git repository before directory", func(t *testing.T) {
		parent := t.TempDir()
		directory := filepath.Join(parent, "repository-name")
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		if output, err := exec.Command("git", "init", "--quiet", directory).CombinedOutput(); err != nil {
			t.Fatalf("git init: %v: %s", err, output)
		}
		var output bytes.Buffer
		if err := setup.Run(setup.Request{Directory: directory, Local: true, Output: &output}); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if got := output.String(); !strings.Contains(got, "Project ID: repository-name") {
			t.Errorf("preview = %q, want Git repository name", got)
		}
	})

	t.Run("explicit override", func(t *testing.T) {
		directory := t.TempDir()
		if err := os.WriteFile(filepath.Join(directory, "package.json"), []byte(`{"name":"detected-name"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		var output bytes.Buffer
		if err := setup.Run(setup.Request{Directory: directory, ProjectID: "chosen-name", Local: true, Output: &output}); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if got := output.String(); !strings.Contains(got, "Project ID: chosen-name") {
			t.Errorf("preview = %q, want explicit project ID", got)
		}
	})

	t.Run("invalid override", func(t *testing.T) {
		directory := t.TempDir()
		credentialStore := store.NewMemory()
		var output bytes.Buffer
		err := setup.Run(setup.Request{Directory: directory, ProjectID: "not valid", Local: true, Confirm: true, Store: credentialStore, Output: &output})
		if err == nil || !strings.Contains(err.Error(), "project_id must be a stable identifier") {
			t.Fatalf("Run() error = %v, want stable identifier error", err)
		}
		if output.Len() != 0 {
			t.Errorf("output = %q, want no invalid preview", output.String())
		}
		if _, err := os.Stat(filepath.Join(directory, "inject.toml")); !os.IsNotExist(err) {
			t.Errorf("inject.toml stat error = %v, want no configuration after invalid input", err)
		}
	})
}

func TestRunPlansBitwardenSource(t *testing.T) {
	directory := t.TempDir()
	var output bytes.Buffer
	err := setup.Run(setup.Request{
		Directory: directory,
		ProjectID: "billing-api",
		Provider:  "bitwarden",
		ItemID:    "note-id",
		Output:    &output,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	preview := output.String()
	if !strings.Contains(preview, "Source choices: Local, 1Password, Bitwarden (selected)") || !strings.Contains(preview, `provider = "bitwarden"`) {
		t.Errorf("preview = %q, want selected Bitwarden source", preview)
	}
	if _, err := os.Stat(filepath.Join(directory, "inject.toml")); !os.IsNotExist(err) {
		t.Errorf("inject.toml stat error = %v, want no configuration without confirmation", err)
	}
}

func TestRunSelectsOnePasswordWhenProviderIsExplicit(t *testing.T) {
	directory := t.TempDir()
	var output bytes.Buffer
	request := request(directory, &output)
	request.Provider = "1password"
	request.Confirm = true
	request.RunValidation = func([]string) error { return nil }

	if err := setup.Run(request); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	config, err := os.ReadFile(filepath.Join(directory, "inject.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(config, []byte("provider = \"1password\"")) {
		t.Errorf("inject.toml = %q, want explicit 1Password source", config)
	}
}

func TestRunImportsLegacyEnvIntoLocalProfile(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, ".env"), []byte("TOKEN=local-value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	credentialStore := store.NewMemory()
	request := setup.Request{
		Directory:     directory,
		ProjectID:     "billing-api",
		Local:         true,
		Confirm:       true,
		Validate:      []string{"go", "test", "./..."},
		RunValidation: func([]string) error { return nil },
		Store:         credentialStore,
		Output:        io.Discard,
	}

	if err := setup.Run(request); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	secrets, err := credentialStore.Get("billing-api", "default")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got, want := secrets["TOKEN"], "local-value"; got != want {
		t.Errorf("stored TOKEN = %q, want %q", got, want)
	}
	config, err := os.ReadFile(filepath.Join(directory, "inject.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(config, []byte("local-value")) || !bytes.Contains(config, []byte("provider = \"local\"")) {
		t.Errorf("inject.toml = %q, want local source without secret values", config)
	}
}

func TestRunDefaultsToLocalMigrationWhenLegacyEnvExists(t *testing.T) {
	directory := t.TempDir()
	envPath := filepath.Join(directory, ".env")
	if err := os.WriteFile(envPath, []byte("TOKEN=local-value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	credentialStore := store.NewMemory()

	if err := setup.Run(setup.Request{
		Directory: directory,
		Confirm:   true,
		Store:     credentialStore,
		Output:    io.Discard,
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	projectID := filepath.Base(directory)
	secrets, err := credentialStore.Get(projectID, "default")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got, want := secrets["TOKEN"], "local-value"; got != want {
		t.Errorf("stored TOKEN = %q, want %q", got, want)
	}
	config, err := os.ReadFile(filepath.Join(directory, "inject.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(config, []byte("local-value")) || !bytes.Contains(config, []byte("project_id = \""+projectID+"\"")) || !bytes.Contains(config, []byte("provider = \"local\"")) {
		t.Errorf("inject.toml = %q, want non-secret default local configuration", config)
	}
	if _, err := os.Stat(envPath); err != nil {
		t.Errorf("legacy .env stat error = %v, want retained", err)
	}
}

func TestRunDefaultLocalMigrationDoesNotWriteConfigurationWhenImportCannotComplete(t *testing.T) {
	for _, test := range []struct {
		name  string
		env   string
		store store.Store
	}{
		{name: "malformed legacy environment", env: "TOKEN\n", store: store.NewMemory()},
		{name: "unavailable credential store", env: "TOKEN=local-value\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			if err := os.WriteFile(filepath.Join(directory, ".env"), []byte(test.env), 0o600); err != nil {
				t.Fatal(err)
			}

			err := setup.Run(setup.Request{
				Directory: directory,
				Confirm:   true,
				Store:     test.store,
				Output:    io.Discard,
			})
			if err == nil {
				t.Fatal("Run() error = nil, want failed local import")
			}
			if _, err := os.Stat(filepath.Join(directory, "inject.toml")); !os.IsNotExist(err) {
				t.Errorf("inject.toml stat error = %v, want no configuration written", err)
			}
		})
	}
}

func TestRunLocalDoesNotImportUntilConfirmed(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, ".env"), []byte("TOKEN=local-value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	credentialStore := store.NewMemory()
	err := setup.Run(setup.Request{
		Directory: directory,
		ProjectID: "billing-api",
		Local:     true,
		Validate:  []string{"true"},
		Store:     credentialStore,
		Output:    io.Discard,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, err := credentialStore.Get("billing-api", "default"); err == nil {
		t.Error("Get() error = nil after cancelled setup, want unavailable secret set")
	}
}

func TestRunLocalValidatesSecretsBeforeConfirmedEnvRemoval(t *testing.T) {
	directory := t.TempDir()
	envPath := filepath.Join(directory, ".env")
	if err := os.WriteFile(envPath, []byte("TOKEN=local-value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := setup.Run(setup.Request{
		Directory:        directory,
		ProjectID:        "billing-api",
		Local:            true,
		Confirm:          true,
		Validate:         []string{"sh", "-c", "test \"$TOKEN\" = local-value"},
		RemoveLegacyEnv:  true,
		ConfirmRemoveEnv: true,
		Store:            store.NewMemory(),
		Output:           io.Discard,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, err := os.Stat(envPath); !os.IsNotExist(err) {
		t.Errorf("legacy .env stat error = %v, want removed after successful validation", err)
	}
}

func TestRunLocalRejectsLegacyEnvRemovalWithoutValidationCommand(t *testing.T) {
	for _, test := range []struct {
		name     string
		validate []string
		wantErr  string
	}{
		{name: "missing command", wantErr: "setup: a finite validation command is required"},
		{name: "non-finite command", validate: []string{"npm", "run", "dev"}, wantErr: "setup: validation command must be finite"},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			envPath := filepath.Join(directory, ".env")
			if err := os.WriteFile(envPath, []byte("TOKEN=local-value\n"), 0o600); err != nil {
				t.Fatal(err)
			}

			err := setup.Run(setup.Request{
				Directory:        directory,
				ProjectID:        "billing-api",
				Local:            true,
				Confirm:          true,
				Validate:         test.validate,
				RemoveLegacyEnv:  true,
				ConfirmRemoveEnv: true,
				Store:            store.NewMemory(),
				Output:           io.Discard,
			})
			if err == nil {
				t.Fatal("Run() error = nil, want finite validation command error")
			}
			if got := err.Error(); got != test.wantErr {
				t.Errorf("Run() error = %q, want %q", got, test.wantErr)
			}
			if _, err := os.Stat(envPath); err != nil {
				t.Errorf("legacy .env stat error = %v, want retained", err)
			}
		})
	}
}

func TestRunLocalDoesNotSaveSecretsWhenValidationFails(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, ".env"), []byte("TOKEN=local-value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	credentialStore := store.NewMemory()
	err := setup.Run(setup.Request{
		Directory:     directory,
		ProjectID:     "billing-api",
		Local:         true,
		Confirm:       true,
		Validate:      []string{"true"},
		RunValidation: func([]string) error { return fmt.Errorf("failed") },
		Store:         credentialStore,
		Output:        io.Discard,
	})
	if err == nil {
		t.Fatal("Run() error = nil, want validation failure")
	}
	if _, err := credentialStore.Get("billing-api", "default"); err == nil {
		t.Error("Get() error = nil after failed validation, want unavailable secret set")
	}
}

func TestRunDoesNotOverwriteExistingConfiguration(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "inject.toml")
	contents := []byte("existing configuration\n")
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	request := request(directory, io.Discard)
	request.Confirm = true
	request.RunValidation = func([]string) error { return nil }
	if err := setup.Run(request); err == nil {
		t.Fatal("Run() error = nil, want existing configuration rejected")
	}
	if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, contents) {
		t.Errorf("inject.toml = %q, %v; want unchanged", got, err)
	}
}

func TestRunRejectsNonFiniteValidationBeforeWritingConfiguration(t *testing.T) {
	directory := t.TempDir()
	request := request(directory, io.Discard)
	request.Confirm = true
	request.Validate = []string{"npm", "run", "dev"}
	request.RunValidation = func([]string) error {
		t.Fatal("RunValidation() was called for a non-finite command")
		return nil
	}

	err := setup.Run(request)
	if err == nil || err.Error() != "setup: validation command must be finite" {
		t.Fatalf("Run() error = %v, want non-finite validation command error", err)
	}
	if _, err := os.Stat(filepath.Join(directory, "inject.toml")); !os.IsNotExist(err) {
		t.Errorf("inject.toml stat error = %v, want no configuration written", err)
	}
}

func TestRunUsesExplicitBindingForUnsafePackageScript(t *testing.T) {
	directory := t.TempDir()
	packagePath := filepath.Join(directory, "package.json")
	contents := []byte(`{"scripts":{"dev":"go run . | tee app.log"}}`)
	if err := os.WriteFile(packagePath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	request := request(directory, io.Discard)
	request.Confirm = true
	request.PackageScript = "dev"
	request.Command = []string{"go", "run", "."}
	request.RunValidation = func([]string) error { return nil }
	err := setup.Run(request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got, err := os.ReadFile(packagePath); err != nil || !bytes.Equal(got, contents) {
		t.Errorf("package.json = %q, %v; want unsafe script unchanged", got, err)
	}
	config, err := os.ReadFile(filepath.Join(directory, "inject.toml"))
	if err != nil || !bytes.Contains(config, []byte("[commands.dev]")) {
		t.Errorf("inject.toml = %q, %v; want explicit dev binding", config, err)
	}
}

func TestRunBindsPackageScriptWithoutChangingPackageManifest(t *testing.T) {
	directory := t.TempDir()
	packagePath := filepath.Join(directory, "package.json")
	contents := []byte(`{"scripts":{"dev":"vite --host 0.0.0.0 | tee app.log"}}`)
	if err := os.WriteFile(packagePath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	request := request(directory, io.Discard)
	request.Confirm = true
	request.PackageScript = "dev"
	request.RunValidation = func([]string) error { return nil }

	if err := setup.Run(request); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got, err := os.ReadFile(packagePath); err != nil || !bytes.Equal(got, contents) {
		t.Errorf("package.json = %q, %v; want unchanged", got, err)
	}
	config, err := os.ReadFile(filepath.Join(directory, "inject.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(config, []byte("[commands.dev]")) || !bytes.Contains(config, []byte(`command = ["npm","run","dev"]`)) {
		t.Errorf("inject.toml = %q, want npm run dev binding", config)
	}
}

func TestRunRejectsUnknownPackageScript(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "package.json"), []byte(`{"scripts":{"dev":"vite"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	request := request(directory, io.Discard)
	request.PackageScript = "test"

	err := setup.Run(request)
	if err == nil || err.Error() != `setup: package.json has no "test" script` {
		t.Fatalf("Run() error = %v, want unknown package script error", err)
	}
}

func TestRunOffersDevAsDefaultPackageScriptBinding(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "package.json"), []byte(`{"scripts":{"test":"go test ./...","dev":"vite","lint":"eslint ."}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	request := request(directory, &output)

	if err := setup.Run(request); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := output.String(); !strings.Contains(got, "Developer command: dev (default)") || !strings.Contains(got, "Developer command: lint") || !strings.Contains(got, "Developer command: test") {
		t.Errorf("output = %q, want dev default and all package scripts", got)
	}
}

func TestRunSelectsDefaultPackageScriptBinding(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "package.json"), []byte(`{"scripts":{"dev":"vite"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	request := request(directory, &output)
	request.SelectPackageScript = func(candidates []string, defaultScript string) (string, error) {
		if want := []string{"dev"}; !reflect.DeepEqual(candidates, want) {
			t.Errorf("candidates = %q, want %q", candidates, want)
		}
		if defaultScript != "dev" {
			t.Errorf("default script = %q, want dev", defaultScript)
		}
		return defaultScript, nil
	}

	if err := setup.Run(request); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := output.String(); !bytes.Contains([]byte(got), []byte("[commands.dev]")) {
		t.Errorf("output = %q, want selected dev binding in preview", got)
	}
}

func TestRunRemovesLegacyEnvOnlyAfterValidationAndSecondConfirmation(t *testing.T) {
	for _, test := range []struct {
		name             string
		confirmRemoval   bool
		wantEnvRemaining bool
	}{
		{name: "without second confirmation", wantEnvRemaining: true},
		{name: "with second confirmation", confirmRemoval: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			envPath := filepath.Join(directory, ".env")
			if err := os.WriteFile(envPath, []byte("TOKEN=secret-value\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			request := request(directory, io.Discard)
			request.Confirm = true
			request.RemoveLegacyEnv = true
			request.ConfirmRemoveEnv = test.confirmRemoval
			request.RunValidation = func(command []string) error {
				if got, want := command, []string{"go", "test", "./..."}; !reflect.DeepEqual(got, want) {
					t.Errorf("validation command = %q, want %q", got, want)
				}
				return nil
			}
			if err := setup.Run(request); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			_, err := os.Stat(envPath)
			if test.wantEnvRemaining && err != nil {
				t.Errorf("legacy .env stat error = %v, want retained", err)
			}
			if !test.wantEnvRemaining && !os.IsNotExist(err) {
				t.Errorf("legacy .env stat error = %v, want removed", err)
			}
		})
	}
}

func request(directory string, output io.Writer) setup.Request {
	return setup.Request{
		Directory: directory,
		ProjectID: "billing-api",
		Account:   "acme",
		Vault:     "Engineering",
		ItemID:    "note-id",
		Validate:  []string{"go", "test", "./..."},
		CheckOnePassword: func() error {
			return nil
		},
		Output: output,
	}
}
