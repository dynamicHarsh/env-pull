// Package store provides project-scoped local secret storage.
package store

import (
	"encoding/json"
	"fmt"
	"time"

	keyring "github.com/zalando/go-keyring"
)

// Store persists complete secret sets under a project ID and profile name.
type Store interface {
	Put(projectID, profile string, secrets map[string]string) error
	Get(projectID, profile string) (map[string]string, error)
	Delete(projectID, profile string) error
	PutCache(projectID, profile string, secrets map[string]string, cachedAt time.Time) error
	GetCache(projectID, profile string) (map[string]string, time.Time, error)
	DeleteCache(projectID, profile string) error
}

// ErrUnavailable reports a missing local secret set without exposing values.
var ErrUnavailable = fmt.Errorf("credential store: secret set unavailable")

// Memory is an in-memory Store intended for tests.
type Memory struct {
	sets   map[string]map[string]map[string]string
	caches map[string]map[string]cachedSecretSet
}

type cachedSecretSet struct {
	Secrets  map[string]string `json:"secrets"`
	CachedAt time.Time         `json:"cached_at"`
}

const serviceName = "inject"

// System stores secret sets in the operating system credential store.
type System struct{}

func NewSystem() *System {
	return &System{}
}

func (store *System) Put(projectID, profile string, secrets map[string]string) error {
	encoded, err := json.Marshal(secrets)
	if err != nil {
		return fmt.Errorf("credential store: encode secret set: %w", err)
	}
	if err := keyring.Set(serviceName, entryName(projectID, profile), string(encoded)); err != nil {
		return fmt.Errorf("credential store: save secret set: %w", err)
	}
	return nil
}

func (store *System) Get(projectID, profile string) (map[string]string, error) {
	encoded, err := keyring.Get(serviceName, entryName(projectID, profile))
	if err != nil {
		return nil, ErrUnavailable
	}
	var secrets map[string]string
	if err := json.Unmarshal([]byte(encoded), &secrets); err != nil || len(secrets) == 0 {
		return nil, ErrUnavailable
	}
	return secrets, nil
}

func (store *System) Delete(projectID, profile string) error {
	if err := keyring.Delete(serviceName, entryName(projectID, profile)); err != nil {
		return fmt.Errorf("credential store: delete secret set: %w", err)
	}
	return nil
}

func (store *System) PutCache(projectID, profile string, secrets map[string]string, cachedAt time.Time) error {
	encoded, err := json.Marshal(cachedSecretSet{Secrets: secrets, CachedAt: cachedAt})
	if err != nil {
		return fmt.Errorf("credential store: encode cache: %w", err)
	}
	if err := keyring.Set(serviceName, cacheEntryName(projectID, profile), string(encoded)); err != nil {
		return fmt.Errorf("credential store: save cache: %w", err)
	}
	return nil
}

func (store *System) GetCache(projectID, profile string) (map[string]string, time.Time, error) {
	encoded, err := keyring.Get(serviceName, cacheEntryName(projectID, profile))
	if err != nil {
		return nil, time.Time{}, ErrUnavailable
	}
	var cached cachedSecretSet
	if err := json.Unmarshal([]byte(encoded), &cached); err != nil || len(cached.Secrets) == 0 || cached.CachedAt.IsZero() {
		return nil, time.Time{}, ErrUnavailable
	}
	return cached.Secrets, cached.CachedAt, nil
}

func (store *System) DeleteCache(projectID, profile string) error {
	if err := keyring.Delete(serviceName, cacheEntryName(projectID, profile)); err != nil {
		return fmt.Errorf("credential store: delete cache: %w", err)
	}
	return nil
}

func NewMemory() *Memory {
	return &Memory{
		sets:   make(map[string]map[string]map[string]string),
		caches: make(map[string]map[string]cachedSecretSet),
	}
}

func (store *Memory) Put(projectID, profile string, secrets map[string]string) error {
	if store.sets[projectID] == nil {
		store.sets[projectID] = make(map[string]map[string]string)
	}
	store.sets[projectID][profile] = copySecrets(secrets)
	return nil
}

func (store *Memory) Get(projectID, profile string) (map[string]string, error) {
	secrets, exists := store.sets[projectID][profile]
	if !exists {
		return nil, ErrUnavailable
	}
	return copySecrets(secrets), nil
}

func (store *Memory) Delete(projectID, profile string) error {
	profiles, exists := store.sets[projectID]
	if !exists {
		return ErrUnavailable
	}
	if _, exists := profiles[profile]; !exists {
		return ErrUnavailable
	}
	delete(profiles, profile)
	if len(profiles) == 0 {
		delete(store.sets, projectID)
	}
	return nil
}

func (store *Memory) PutCache(projectID, profile string, secrets map[string]string, cachedAt time.Time) error {
	if store.caches[projectID] == nil {
		store.caches[projectID] = make(map[string]cachedSecretSet)
	}
	store.caches[projectID][profile] = cachedSecretSet{Secrets: copySecrets(secrets), CachedAt: cachedAt}
	return nil
}

func (store *Memory) GetCache(projectID, profile string) (map[string]string, time.Time, error) {
	cached, exists := store.caches[projectID][profile]
	if !exists {
		return nil, time.Time{}, ErrUnavailable
	}
	return copySecrets(cached.Secrets), cached.CachedAt, nil
}

func (store *Memory) DeleteCache(projectID, profile string) error {
	profiles, exists := store.caches[projectID]
	if !exists {
		return ErrUnavailable
	}
	if _, exists := profiles[profile]; !exists {
		return ErrUnavailable
	}
	delete(profiles, profile)
	if len(profiles) == 0 {
		delete(store.caches, projectID)
	}
	return nil
}

func copySecrets(secrets map[string]string) map[string]string {
	copy := make(map[string]string, len(secrets))
	for key, value := range secrets {
		copy[key] = value
	}
	return copy
}

func entryName(projectID, profile string) string {
	return projectID + ":" + profile
}

func cacheEntryName(projectID, profile string) string {
	return "cache:" + entryName(projectID, profile)
}
