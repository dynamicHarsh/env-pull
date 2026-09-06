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
	setupPackageScripts   []string
	setupCommand          []string
	setupValidation       []string
	setupLocal            bool
	setupSelectedInputs   []string
	setupConfirm          bool
	setupRemoveLegacyEnv  bool
	setupConfirmRemoveEnv bool
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure inject for a local or remote secret source",
	Long: `setup previews a non-secret inject.toml configuration and optional
command bindings. Selected package scripts are preserved under inject-owned names
so their existing package-manager commands continue to work. Setup validates the selected
provider through its existing CLI authentication context before changing project files.
A detected .env selects local credential-store
migration unless a remote provider or reference is selected.

Use --yes to apply the preview. Remote setup requires a finite validation command.
Use --remove-env together with --yes-remove-env to delete a detected legacy .env
after a successful validation.`,
	Args: cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		interactive := isTerminal(os.Stdin) && isTerminal(os.Stdout)
		var scriptSelector func([]string, string) (string, error)
		if interactive {
			scriptSelector = selectPackageScript
		}
		return setupworkflow.Run(setupworkflow.Request{
			ProjectID:           setupProjectID,
			Provider:            setupProvider,
			Account:             setupAccount,
			Vault:               setupVault,
			ItemID:              setupItemID,
			Item:                setupItem,
			Binding:             setupBinding,
			PackageScripts:      setupPackageScripts,
			SelectPackageScript: scriptSelector,
			Command:             setupCommand,
			Validate:            setupValidation,
			Local:               setupLocal,
			SelectedInputs:      setupSelectedInputs,
			Store:               store.NewSystem(),
			Confirm:             setupConfirm,
			RemoveLegacyEnv:     setupRemoveLegacyEnv,
			ConfirmRemoveEnv:    setupConfirmRemoveEnv,
			NonInteractive:      !interactive,
			Output:              os.Stdout,
		})
	},
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
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
	setupCmd.Flags().StringVar(&setupProvider, "provider", "", "secret provider (1password or bitwarden)")
	setupCmd.Flags().StringVar(&setupAccount, "account", "", "1Password account")
	setupCmd.Flags().StringVar(&setupVault, "vault", "", "1Password vault")
	setupCmd.Flags().StringVar(&setupItemID, "item-id", "", "immutable remote item ID")
	setupCmd.Flags().StringVar(&setupItem, "item", "", "remote item name")
	setupCmd.Flags().StringVar(&setupBinding, "binding", "", "explicit command binding name")
	setupCmd.Flags().StringSliceVar(&setupPackageScripts, "package-script", nil, "package.json script to preserve through injection; repeat for each script")
	setupCmd.Flags().StringArrayVar(&setupCommand, "command", nil, "binding command argument; repeat for each argument")
	setupCmd.Flags().StringArrayVar(&setupValidation, "validate", nil, "finite validation command argument; repeat for each argument")
	setupCmd.Flags().BoolVar(&setupLocal, "local", false, "force import of a legacy .env into the local credential store")
	setupCmd.Flags().StringSliceVar(&setupSelectedInputs, "env-file", nil, "plaintext environment input to import; repeat for variants")
	setupCmd.Flags().BoolVar(&setupConfirm, "yes", false, "apply the previewed project changes")
	setupCmd.Flags().BoolVar(&setupRemoveLegacyEnv, "remove-env", false, "remove a legacy .env after validation")
	setupCmd.Flags().BoolVar(&setupConfirmRemoveEnv, "yes-remove-env", false, "confirm removal of a legacy .env")
	rootCmd.AddCommand(setupCmd)
}
