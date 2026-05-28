package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/harsh-sonkar/env-pull/internal/crypto"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Decrypt the local vault and print it to stdout",
	Long: `export decrypts ` + defaultVaultFile + ` and writes the plaintext to stdout.

The intended usage is to pipe the output into a file:

  env-pull export > .env

This is an escape hatch — if you decide env-pull is not for you, this
command lets you recover a standard plaintext .env file at any time.
No credentials are held hostage.

Nothing is written to disk by env-pull itself; the shell redirection
(> .env) is entirely under your control.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat(defaultVaultFile); os.IsNotExist(err) {
			return fmt.Errorf("export: no local vault found (%s does not exist)", defaultVaultFile)
		}

		key, err := crypto.LoadOrCreateKey()
		if err != nil {
			return fmt.Errorf("export: %w", err)
		}
		defer crypto.ZeroBytes(key)

		ciphertext, err := os.ReadFile(defaultVaultFile)
		if err != nil {
			return fmt.Errorf("export: failed to read vault: %w", err)
		}

		plaintext, err := crypto.Decrypt(ciphertext, key)
		if err != nil {
			return fmt.Errorf("export: %w", err)
		}
		defer crypto.ZeroBytes(plaintext)

		_, err = os.Stdout.Write(plaintext)
		return err
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
}
