// Package store provides project-scoped local secret storage.
package store

import (
	"encoding/json"
	"fmt"

	keyring "github.com/zalando/go-keyring"
)

// Store persists complete secret sets under a project ID and profile name.
type Store interface {
	Put(projectID, profile string, secrets map[string]string) error
	Get(projectID, profile string) (map[string]string, error)
}

// ErrUnavailable reports a missing local secret set without exposing values.
var ErrUnavailable = fmt.Errorf("credential store: secret set unavailable")

// Memory is an in-memory Store intended for tests.
type Memory struct {
	sets map[string]map[string]map[string]string
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

func NewMemory() *Memory {
	return &Memory{sets: make(map[string]map[string]map[string]string)}
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
