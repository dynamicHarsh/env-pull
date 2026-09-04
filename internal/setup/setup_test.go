package setup_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/harsh-sonkar/env-pull/internal/setup"
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
