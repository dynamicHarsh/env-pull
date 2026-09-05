package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	setupworkflow "github.com/harsh-sonkar/env-pull/internal/setup"
	"github.com/harsh-sonkar/env-pull/internal/store"
)

var (
	setupProjectID        string
	setupProvider         string
	setupAccount          string
	setupVault            string
	setupItemID           string
	setupItem             string
	setupBinding          string
	setupPackageScript    string
	setupCommand          []string
	setupValidation       []string
	setupLocal            bool
	setupConfirm          bool
	setupRemoveLegacyEnv  bool
	setupConfirmRemoveEnv bool
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure inject for a 1Password secret note",
	Long: `setup previews a non-secret inject.toml configuration and optional
command binding. Package-script bindings run as npm run <script> without changing
package.json. It checks the existing op CLI session before changing
project files for remote sources. A detected .env selects local credential-store
migration unless a remote provider or reference is selected.

Use --yes to apply the preview. Remote setup requires a finite validation command.
Use --remove-env together with --yes-remove-env to delete a detected legacy .env
after a successful validation.`,
	Args: cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		return setupworkflow.Run(setupworkflow.Request{
			ProjectID:           setupProjectID,
			Provider:            setupProvider,
			Account:             setupAccount,
			Vault:               setupVault,
			ItemID:              setupItemID,
			Item:                setupItem,
			Binding:             setupBinding,
			PackageScript:       setupPackageScript,
			SelectPackageScript: selectPackageScript,
			Command:             setupCommand,
			Validate:            setupValidation,
			Local:               setupLocal,
			Store:               store.NewSystem(),
			Confirm:             setupConfirm,
			RemoveLegacyEnv:     setupRemoveLegacyEnv,
			ConfirmRemoveEnv:    setupConfirmRemoveEnv,
			Output:              os.Stdout,
		})
	},
}

func selectPackageScript(candidates []string, defaultScript string) (string, error) {
	if len(candidates) == 0 {
		return "", nil
	}
	if defaultScript == "" {
		fmt.Fprint(os.Stdout, "Select a package script (or none): ")
	} else {
		fmt.Fprintf(os.Stdout, "Select a package script [%s] (or none): ", defaultScript)
	}
	selection, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("setup: read package script selection: %w", err)
	}
	selection = strings.TrimSpace(selection)
	if selection == "" {
		return defaultScript, nil
	}
	if selection == "none" {
		return "", nil
	}
	for _, candidate := range candidates {
		if selection == candidate {
			return selection, nil
		}
	}
	return "", fmt.Errorf("setup: package script %q is not available", selection)
}

func init() {
	setupCmd.Flags().StringVar(&setupProjectID, "project-id", "", "stable project identifier")
	setupCmd.Flags().StringVar(&setupProvider, "provider", "", "secret provider (1password)")
	setupCmd.Flags().StringVar(&setupAccount, "account", "", "1Password account")
	setupCmd.Flags().StringVar(&setupVault, "vault", "", "1Password vault")
	setupCmd.Flags().StringVar(&setupItemID, "item-id", "", "immutable 1Password item ID")
	setupCmd.Flags().StringVar(&setupItem, "item", "", "1Password item name")
	setupCmd.Flags().StringVar(&setupBinding, "binding", "", "explicit command binding name")
	setupCmd.Flags().StringVar(&setupPackageScript, "package-script", "", "package.json script to bind as npm run <script>")
	setupCmd.Flags().StringArrayVar(&setupCommand, "command", nil, "binding command argument; repeat for each argument")
	setupCmd.Flags().StringArrayVar(&setupValidation, "validate", nil, "finite validation command argument; repeat for each argument")
	setupCmd.Flags().BoolVar(&setupLocal, "local", false, "force import of a legacy .env into the local credential store")
	setupCmd.Flags().BoolVar(&setupConfirm, "yes", false, "apply the previewed project changes")
	setupCmd.Flags().BoolVar(&setupRemoveLegacyEnv, "remove-env", false, "remove a legacy .env after validation")
	setupCmd.Flags().BoolVar(&setupConfirmRemoveEnv, "yes-remove-env", false, "confirm removal of a legacy .env")
	rootCmd.AddCommand(setupCmd)
}
