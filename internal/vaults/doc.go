// Package vaults defines the Vault interface and provides concrete adapters
// for upstream secret providers (e.g., AWS Secrets Manager, 1Password).
// Each adapter implements the Vault interface and follows a zero-config
// approach: it reuses the developer's existing authentication context
// (e.g., ~/.aws/credentials, 1Password IPC socket) rather than requiring
// additional tokens or login flows.
package vaults
