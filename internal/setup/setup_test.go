package setup_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
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
	if got, want := output.String(), "1Password is unavailable; run `op signin` and retry\n"; got != want {
		t.Errorf("output = %q, want %q", got, want)
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

func TestRunSelectsOnePasswordWhenProviderIsExplicit(t *testing.T) {
	directory := t.TempDir()
	request := request(directory, io.Discard)
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
