package cmd

import (
	"fmt"
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
	RunE: func(_ *cobra.Command, _ []string) error {
		if !removeConfirm {
			return fmt.Errorf("remove: pass --yes to remove this project")
		}
		return removeProject(".", store.NewSystem())
	},
}

func removeProject(directory string, credentialStore store.Store) error {
	configPath := filepath.Join(directory, project.FileName)
	config, err := project.Load(configPath)
	if err != nil {
		return fmt.Errorf("remove: %w", err)
	}
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
		return fmt.Errorf("remove: delete inject.toml: %w", err)
	}
	return nil
}

func init() {
	removeCmd.Flags().BoolVar(&removeConfirm, "yes", false, "confirm project removal")
	rootCmd.AddCommand(removeCmd)
}
