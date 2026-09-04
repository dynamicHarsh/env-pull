package store_test

import (
	"reflect"
	"testing"

	"github.com/harsh-sonkar/env-pull/internal/store"
)

func TestMemoryStoreScopesSecretSetsByProjectAndProfile(t *testing.T) {
	credentialStore := store.NewMemory()
	want := map[string]string{"TOKEN": "local-value"}

	if err := credentialStore.Put("billing-api", "default", want); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	got, err := credentialStore.Get("billing-api", "default")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Get() = %q, want %q", got, want)
	}
	if _, err := credentialStore.Get("other-project", "default"); err == nil {
		t.Error("Get() error = nil for another project, want unavailable secret set")
	}
	if _, err := credentialStore.Get("billing-api", "staging"); err == nil {
		t.Error("Get() error = nil for another profile, want unavailable secret set")
	}
}

func TestMemoryStoreDeletesOnlyRequestedSecretSet(t *testing.T) {
	credentialStore := store.NewMemory()
	if err := credentialStore.Put("billing-api", "default", map[string]string{"TOKEN": "local-value"}); err != nil {
		t.Fatal(err)
	}
	if err := credentialStore.Put("billing-api", "staging", map[string]string{"TOKEN": "staging-value"}); err != nil {
		t.Fatal(err)
	}

	if err := credentialStore.Delete("billing-api", "default"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := credentialStore.Get("billing-api", "default"); err == nil {
		t.Error("deleted profile remains available")
	}
	if _, err := credentialStore.Get("billing-api", "staging"); err != nil {
		t.Errorf("other profile was deleted: %v", err)
	}
}
