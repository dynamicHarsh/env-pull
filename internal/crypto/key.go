package crypto

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// LoadOrCreateKey reads the 32-byte master key from ~/.config/env-pull/master.key.
// If the file does not exist the key directory is created with 0700 permissions,
// a fresh 32-byte random key is generated, saved with 0600 permissions, and returned.
// The caller is responsible for zeroing the returned slice after use.
func LoadOrCreateKey() ([]byte, error) {
	path, err := masterKeyPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err == nil {
		if len(data) != 32 {
			return nil, fmt.Errorf("key: master key at %s has invalid length %d (expected 32 bytes)", path, len(data))
		}
		return data, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("key: failed to read master key: %w", err)
	}

	return generateAndSaveKey(path)
}

func masterKeyPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("key: failed to resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "env-pull", "master.key"), nil
}

func generateAndSaveKey(path string) ([]byte, error) {
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("key: failed to create config directory: %w", err)
	}
	// MkdirAll honours the process umask, which can widen permissions.
	// Explicitly tighten to 0700 on POSIX systems.
	if runtime.GOOS != "windows" {
		if err := os.Chmod(dir, 0700); err != nil {
			return nil, fmt.Errorf("key: failed to restrict directory permissions: %w", err)
		}
	}

	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("key: failed to generate random key: %w", err)
	}

	if err := os.WriteFile(path, key, 0600); err != nil {
		zeroBytes(key)
		return nil, fmt.Errorf("key: failed to save master key: %w", err)
	}

	return key, nil
}
