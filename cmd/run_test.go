package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/harsh-sonkar/env-pull/internal/project"
	"github.com/harsh-sonkar/env-pull/internal/store"
)

func TestLoadLocalProfileSecretsUsesProjectAndProfileScope(t *testing.T) {
	credentialStore := store.NewMemory()
	if err := credentialStore.Put("billing-api", "staging", map[string]string{"TOKEN": "staging-value"}); err != nil {
		t.Fatal(err)
	}
	config := project.Config{
		ProjectID: "billing-api",
		Profiles:  map[string]project.Profile{"staging": {Provider: "local"}},
	}

	got, err := loadLocalProfileSecrets(config, "staging", credentialStore)
	if err != nil {
		t.Fatalf("loadLocalProfileSecrets() error = %v", err)
	}
	if want := map[string]string{"TOKEN": "staging-value"}; !reflect.DeepEqual(got, want) {
		t.Errorf("loadLocalProfileSecrets() = %q, want %q", got, want)
	}
}

func TestRemoveProjectDeletesLocalProfilesAndConfiguration(t *testing.T) {
	directory := t.TempDir()
	config := `format_version = 1
project_id = "billing-api"

[profiles.default]
provider = "local"

[profiles.staging]
provider = "local"
`
	if err := os.WriteFile(filepath.Join(directory, project.FileName), []byte(config), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	credentialStore := store.NewMemory()
	for _, profileName := range []string{"default", "staging"} {
		if err := credentialStore.Put("billing-api", profileName, map[string]string{"TOKEN": profileName}); err != nil {
			t.Fatal(err)
		}
	}

	if err := removeProject(directory, credentialStore); err != nil {
		t.Fatalf("removeProject() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, project.FileName)); !os.IsNotExist(err) {
		t.Errorf("inject.toml stat error = %v, want deleted", err)
	}
	for _, profileName := range []string{"default", "staging"} {
		if _, err := credentialStore.Get("billing-api", profileName); err == nil {
			t.Errorf("local profile %q remains available", profileName)
		}
	}
}
