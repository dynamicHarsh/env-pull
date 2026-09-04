package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

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

func TestLoadRemoteProfileSecretsRefreshesOptInCacheForOfflineUse(t *testing.T) {
	credentialStore := store.NewMemory()
	now := time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)
	config := project.Config{
		ProjectID: "billing-api",
		Cache:     project.CachePolicy{Enabled: true, MaxAge: project.Duration{Duration: time.Hour}},
		Profiles:  map[string]project.Profile{"staging": {Provider: "bitwarden"}},
	}
	fetches := 0
	fetch := func() (map[string]string, error) {
		fetches++
		return map[string]string{"TOKEN": "fresh-value"}, nil
	}

	got, err := loadRemoteProfileSecrets(config, "staging", false, credentialStore, false, now, fetch)
	if err != nil {
		t.Fatalf("fresh load error = %v", err)
	}
	if want := map[string]string{"TOKEN": "fresh-value"}; !reflect.DeepEqual(got, want) {
		t.Errorf("fresh load = %q, want %q", got, want)
	}

	got, err = loadRemoteProfileSecrets(config, "staging", true, credentialStore, false, now.Add(30*time.Minute), fetch)
	if err != nil {
		t.Fatalf("offline load error = %v", err)
	}
	if want := map[string]string{"TOKEN": "fresh-value"}; !reflect.DeepEqual(got, want) {
		t.Errorf("offline load = %q, want %q", got, want)
	}
	if fetches != 1 {
		t.Errorf("fetches = %d, want 1", fetches)
	}
}

func TestLoadRemoteProfileSecretsRejectsUnavailableOfflineCaches(t *testing.T) {
	now := time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)
	cachePolicy := project.CachePolicy{Enabled: true, MaxAge: project.Duration{Duration: time.Hour}}
	tests := []struct {
		name      string
		cache     project.CachePolicy
		profile   string
		cachedAt  time.Time
		cacheName string
		ci        bool
	}{
		{name: "caching disabled", cache: project.CachePolicy{}},
		{name: "cache missing", cache: cachePolicy},
		{name: "cache expired", cache: cachePolicy, cachedAt: now.Add(-time.Hour), cacheName: "default"},
		{name: "other profile cache", cache: cachePolicy, cachedAt: now, cacheName: "staging", profile: "default"},
		{name: "CI", cache: cachePolicy, cachedAt: now, cacheName: "default", ci: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			credentialStore := store.NewMemory()
			if test.cacheName != "" {
				if err := credentialStore.PutCache("billing-api", test.cacheName, map[string]string{"TOKEN": "cached-value"}, test.cachedAt); err != nil {
					t.Fatal(err)
				}
			}
			fetches := 0
			_, err := loadRemoteProfileSecrets(project.Config{ProjectID: "billing-api", Cache: test.cache}, test.profile, true, credentialStore, test.ci, now, func() (map[string]string, error) {
				fetches++
				return map[string]string{"TOKEN": "fresh-value"}, nil
			})
			if err == nil {
				t.Fatal("offline load error = nil, want unavailable cache error")
			}
			if fetches != 0 {
				t.Errorf("fetches = %d, want 0", fetches)
			}
		})
	}
}

func TestLoadRemoteProfileSecretsDoesNotAccessCacheInCI(t *testing.T) {
	credentialStore := &cacheAccessStore{Store: store.NewMemory()}
	config := project.Config{
		ProjectID: "billing-api",
		Cache:     project.CachePolicy{Enabled: true, MaxAge: project.Duration{Duration: time.Hour}},
	}
	now := time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)
	if _, err := loadRemoteProfileSecrets(config, "default", false, credentialStore, true, now, func() (map[string]string, error) {
		return map[string]string{"TOKEN": "fresh-value"}, nil
	}); err != nil {
		t.Fatalf("fresh CI load error = %v", err)
	}
	if _, err := loadRemoteProfileSecrets(config, "default", true, credentialStore, true, now, func() (map[string]string, error) {
		t.Fatal("offline CI load fetched remote secrets")
		return nil, nil
	}); err == nil {
		t.Fatal("offline CI load error = nil, want unavailable cache error")
	}
	if credentialStore.cacheReads != 0 || credentialStore.cacheWrites != 0 {
		t.Errorf("cache reads/writes = %d/%d, want 0/0", credentialStore.cacheReads, credentialStore.cacheWrites)
	}
}

func TestLoadProfileSecretsCachesBothRemoteProvidersForOfflineUse(t *testing.T) {
	directory := t.TempDir()
	originalDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDirectory) })

	config := `format_version = 1
project_id = "billing-api"

[cache]
enabled = true

[profiles.onepassword]
provider = "1password"
account = "acme"
vault = "Engineering"
item_id = "onepassword-note"

[profiles.bitwarden]
provider = "bitwarden"
item_id = "bitwarden-note"
`
	if err := os.WriteFile(project.FileName, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("op", []byte("#!/bin/sh\nprintf '%s\\n' '{\"fields\":[{\"id\":\"notesPlain\",\"value\":\"TOKEN=from-onepassword\\n\"}]}'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("bw", []byte("#!/bin/sh\nprintf 'TOKEN=from-bitwarden\\n'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	credentialStore := store.NewMemory()
	originalStoreFactory := newCredentialStore
	newCredentialStore = func() store.Store { return credentialStore }
	t.Cleanup(func() { newCredentialStore = originalStoreFactory })
	t.Setenv("PATH", directory)

	for _, profileName := range []string{"onepassword", "bitwarden"} {
		fresh, err := loadProfileSecrets(profileName, false)
		if err != nil {
			t.Fatalf("fresh %s load: %v", profileName, err)
		}
		t.Setenv("PATH", t.TempDir())
		offline, err := loadProfileSecrets(profileName, true)
		if err != nil {
			t.Fatalf("offline %s load: %v", profileName, err)
		}
		if !reflect.DeepEqual(offline, fresh) {
			t.Errorf("offline %s secrets = %q, want %q", profileName, offline, fresh)
		}
		t.Setenv("PATH", directory)
	}
}

func TestExtractRunFlags(t *testing.T) {
	profile, offline, remaining := extractRunFlags([]string{"--profile=staging", "--offline", "--", "sh", "-c", "echo ok"})
	if profile != "staging" || !offline {
		t.Errorf("flags = profile %q, offline %t; want staging, true", profile, offline)
	}
	if want := []string{"--", "sh", "-c", "echo ok"}; !reflect.DeepEqual(remaining, want) {
		t.Errorf("remaining = %q, want %q", remaining, want)
	}
}

type cacheAccessStore struct {
	store.Store
	cacheReads  int
	cacheWrites int
}

func (store *cacheAccessStore) PutCache(projectID, profile string, secrets map[string]string, cachedAt time.Time) error {
	store.cacheWrites++
	return store.Store.PutCache(projectID, profile, secrets, cachedAt)
}

func (store *cacheAccessStore) GetCache(projectID, profile string) (map[string]string, time.Time, error) {
	store.cacheReads++
	return store.Store.GetCache(projectID, profile)
}

func TestRemoveProjectDeletesLocalProfilesAndConfiguration(t *testing.T) {
	directory := t.TempDir()
	config := `format_version = 1
project_id = "billing-api"

[profiles.default]
provider = "local"

[profiles.staging]
provider = "local"

[profiles.remote]
provider = "bitwarden"
item_id = "stable-note-id"
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
	if err := credentialStore.PutCache("billing-api", "remote", map[string]string{"TOKEN": "cached-value"}, time.Now()); err != nil {
		t.Fatal(err)
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
	if _, _, err := credentialStore.GetCache("billing-api", "remote"); err == nil {
		t.Error("remote cache remains available")
	}
}
