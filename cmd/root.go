// Package cmd contains the Cobra command definitions for inject.
// Command handlers are intentionally thin: they parse flags and delegate
// all business logic to packages under internal/.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/harsh-sonkar/env-pull/internal/project"
)

// rootCmd is the base command for the inject CLI.
//
// inject is a Universal Secrets Adapter. It fetches credentials from an
// upstream enterprise vault (AWS Secrets Manager, 1Password, etc.) or from a
// locally encrypted file managed by env-edit, then spawns the caller's target
// command as a child process with those secrets injected via OS-level
// environment inheritance. Secrets are held only in process memory and are
// never written to disk or emitted to any log.
var rootCmd = &cobra.Command{
	Use:   "inject",
	Short: "Inject secrets into a foreground child process",
	Long: `inject supplies a secret set only to a foreground child process.

Secrets are held in memory while the child runs and are not exported to the
invoking shell. Use inject run for one-off commands.`,
}

// Execute is the single entry point called by main. It runs the root command
// and exits with a non-zero status code on any error.
func Execute() {
	registerBindings()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func registerBindings() {
	config, err := project.Find()
	if err != nil {
		return
	}
	for name, binding := range config.Commands {
		binding := binding
		rootCmd.AddCommand(&cobra.Command{
			Use:   name,
			Short: "Run the configured " + name + " command",
			Args:  cobra.NoArgs,
			RunE: func(_ *cobra.Command, _ []string) error {
				secrets, err := loadProfileSecrets(binding.Profile)
				if err != nil {
					return err
				}
				if err := runChild(binding.Command, secrets); err != nil {
					return fmt.Errorf("%s: %w", name, err)
				}
				return nil
			},
		})
	}
}
