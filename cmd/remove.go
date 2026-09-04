package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/harsh-sonkar/env-pull/internal/project"
	"github.com/harsh-sonkar/env-pull/internal/store"
)

var removeConfirm bool

var removeCmd = &cobra.Command{
	Use:   "remove --yes",
	Short: "Remove this project's local configuration and credentials",
	Args:  cobra.NoArgs,
	RunE: func(command *cobra.Command, _ []string) error {
		return runRemove(".", store.NewSystem(), removeConfirm, command.OutOrStdout())
	},
}

func runRemove(directory string, credentialStore store.Store, confirmed bool, output io.Writer) error {
	configPath := filepath.Join(directory, project.FileName)
	config, err := project.Load(configPath)
	if err != nil {
		return fmt.Errorf("remove: %w", err)
	}
	if err := previewRemoval(configPath, config, credentialStore, output); err != nil {
		return err
	}
	if !confirmed {
		return fmt.Errorf("remove: pass --yes to remove this project")
	}
	return removeConfig(configPath, config, credentialStore)
}

func previewRemoval(configPath string, config project.Config, credentialStore store.Store, output io.Writer) error {
	fmt.Fprintln(output, "The following local state will be removed:")
	fmt.Fprintf(output, "- configuration %q\n", configPath)
	for profileName, profile := range config.Profiles {
		if profile.Provider == "local" {
			if _, err := credentialStore.Get(config.ProjectID, profileName); err == nil {
				fmt.Fprintf(output, "- local credential-store entry for profile %q\n", profileName)
			} else if err != store.ErrUnavailable {
				return fmt.Errorf("remove: inspect local profile %q: %w", profileName, err)
			}
			continue
		}
		if _, _, err := credentialStore.GetCache(config.ProjectID, profileName); err == nil {
			fmt.Fprintf(output, "- remote cache for profile %q\n", profileName)
		} else if err != store.ErrUnavailable {
			return fmt.Errorf("remove: inspect remote cache for profile %q: %w", profileName, err)
		}
	}
	return nil
}

func removeConfig(configPath string, config project.Config, credentialStore store.Store) error {
	for profileName, profile := range config.Profiles {
		if profile.Provider == "local" {
			if err := credentialStore.Delete(config.ProjectID, profileName); err != nil && err != store.ErrUnavailable {
				return fmt.Errorf("remove: delete local profile %q: %w", profileName, err)
			}
			continue
		}
		if err := credentialStore.DeleteCache(config.ProjectID, profileName); err != nil && err != store.ErrUnavailable {
			return fmt.Errorf("remove: delete remote cache for profile %q: %w", profileName, err)
		}
	}
	if err := os.Remove(configPath); err != nil {
		return fmt.Errorf("remove: delete %s: %w", project.FileName, err)
	}
	return nil
}

func init() {
	removeCmd.Flags().BoolVar(&removeConfirm, "yes", false, "confirm project removal")
	rootCmd.AddCommand(removeCmd)
}
