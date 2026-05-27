package vaults

import "context"

// Provider is the common interface for all secret backends.
// Implementations must fetch the named secret and return its contents as a
// flat map of environment variable names to their plaintext values.
// The caller is responsible for letting the map fall out of scope promptly;
// individual string values cannot be zeroed in Go without unsafe.
type Provider interface {
	Fetch(ctx context.Context, secretName string) (map[string]string, error)
}
