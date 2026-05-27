// Package cmd contains the Cobra command definitions for env-pull.
// Command handlers are intentionally thin: they parse flags and delegate
// all business logic to packages under internal/.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the base command for the env-pull CLI.
//
// env-pull is a Universal Secrets Adapter. It fetches credentials from an
// upstream enterprise vault (AWS Secrets Manager, 1Password, etc.) or from a
// locally encrypted file managed by env-edit, then spawns the caller's target
// command as a child process with those secrets injected via OS-level
// environment inheritance. Secrets are held only in process memory and are
// never written to disk or emitted to any log.
var rootCmd = &cobra.Command{
	Use:   "env-pull",
	Short: "Universal Secrets Adapter — inject vault secrets into any process",
	Long: `env-pull is a Universal Secrets Adapter CLI.

THE PROBLEM
  Traditional .env files sit on developer laptops as plaintext, get committed
  to git by accident, and travel over Slack. Every rotation means updating
  every copy on every machine.

THE SOLUTION — ZERO-DISK INJECTION
  env-pull fetches credentials from a vault at runtime, spawns your command
  as a child process with those secrets present in its environment via
  OS-level process tree inheritance, then lets them vanish when the process
  exits. Secrets never touch the filesystem as plaintext and are never logged.

TWO BACKENDS, ZERO ADDITIONAL CONFIGURATION

  Local encrypted vault (team-friendly, offline-capable):
    env-pull edit               — open and save secrets in AES-256-GCM vault
    env-pull run -- <cmd>       — inject vault secrets into any command

  Upstream vault (zero-config, reuses existing AWS credentials):
    env-pull run --aws-secret <name> -- <cmd>
                                — fetch from AWS Secrets Manager and inject

ARCHITECTURE
  1. env-pull loads the secret source (vault file or AWS Secrets Manager).
  2. Secrets are held only in process memory — never written to disk.
  3. A child process is spawned with secrets in its environment.
  4. The child exits; its environment is reclaimed by the OS automatically.`,
}

// Execute is the single entry point called by main. It runs the root command
// and exits with a non-zero status code on any error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
