package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/harsh-sonkar/env-pull/internal/crypto"
)

const defaultVaultFile = ".env.pull.enc"

// editCmd is intentionally thin: it delegates all encryption, editor-launch,
// and secure-deletion logic to crypto.OpenInEditor.
var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open the encrypted secrets vault in your default editor",
	Long: `edit manages your local encrypted secrets vault.

It decrypts ` + defaultVaultFile + `, opens it in your preferred editor,
then re-encrypts the result and saves it back when you exit. Write your
secrets in standard .env format, one KEY=VALUE pair per line:

  DB_PASSWORD=s3cr3t
  API_KEY=abc123
	# This is a comment — ignored by inject run

ZERO-DISK GUARANTEE
  The plaintext is written to a temporary file only for the duration of your
	editing session. When the editor exits, inject overwrites that file with
  zeros before deleting it, limiting the chance of plaintext lingering on disk.

EDITOR SELECTION
	inject respects the $EDITOR environment variable. If it is not set, vim
  is used on Linux/macOS and notepad.exe on Windows. Multi-word values are
  supported (e.g. EDITOR="code --wait").

KEY MANAGEMENT
  Vault file : ` + defaultVaultFile + `  (AES-256-GCM encrypted, created on first use)
  Master key : ~/.config/env-pull/master.key  (32 random bytes, 0600 permissions)

  The master key is generated automatically the first time you run 'edit'.
  Back it up securely — without it the vault cannot be decrypted.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := crypto.OpenInEditor(defaultVaultFile); err != nil {
			return fmt.Errorf("edit: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
