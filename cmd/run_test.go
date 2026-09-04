package cmd

import (
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