package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/harsh-sonkar/env-pull/internal/project"
)

var packageScriptCmd = &cobra.Command{
	Use:    "__run-package-script <name>",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		config, err := project.Find()
		if err != nil {
			return fmt.Errorf("package script: %w", err)
		}
		binding, err := config.ScriptBinding(args[0])
		if err != nil {
			return fmt.Errorf("package script: %w", err)
		}
		secrets, err := loadProfileSecrets(binding.Profile, false)
		if err != nil {
			return err
		}
		manager := invokingPackageManager(binding.PackageManager)
		for _, script := range []string{binding.PreScript, binding.Script, binding.PostScript} {
			if script == "" {
				continue
			}
			if err := runChild([]string{manager, "run", script}, secrets); err != nil {
				return fmt.Errorf("package script %q: %w", args[0], err)
			}
		}
		return nil
	},
}

func invokingPackageManager(fallback string) string {
	if executable := os.Getenv("npm_execpath"); executable != "" {
		if info, err := os.Stat(executable); err == nil && info.Mode().IsRegular() && info.Mode()&0o111 != 0 {
			return executable
		}
	}
	if family, _, found := strings.Cut(os.Getenv("npm_config_user_agent"), "/"); found {
		switch family {
		case "npm", "pnpm", "yarn", "bun":
			if executable, err := exec.LookPath(family); err == nil {
				return executable
			}
		}
	}
	return fallback
}

func init() {
	rootCmd.AddCommand(packageScriptCmd)
}
