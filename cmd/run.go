package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/harsh-sonkar/env-pull/internal/executor"
	"github.com/harsh-sonkar/env-pull/internal/project"
	"github.com/harsh-sonkar/env-pull/internal/vaults"
)

// runCmd is intentionally thin: it resolves secrets from the appropriate
// backend, then delegates all execution to executor.RunCommand.
var runCmd = &cobra.Command{
	Use:   "run [--profile <name>] -- <command> [args...]",
	Short: "Run a command with configured secrets injected into its environment",
	Long: `run is the core command of inject. It resolves secrets from inject.toml,
then spawns your target command as a transparent child process with those
secrets present in its environment. The child sees them as ordinary env vars;
they vanish when it exits. Nothing is written to disk.

The default profile is selected unless --profile names another configured
1Password secret note. inject reuses the existing op CLI session; run
op signin before retrying when it reports an unavailable session.

USAGE EXAMPLES

	inject run -- ./server

	# Run with a named profile
	inject run --profile staging -- psql --host=localhost mydb

  # Works with any program, including shell built-ins via a shell wrapper
	inject run -- printenv DB_PASSWORD

NOTE: all inject flags must appear before the child command. Arguments
after the child binary name are passed through unchanged, including flags
that look like -x or --flag (flag parsing is disabled for this subcommand).`,

	// DisableFlagParsing passes all arguments through to the child command
	// unchanged. Our own --aws-secret flag is extracted manually before dispatch.
	DisableFlagParsing: true,

	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("run: a command to execute is required")
		}

		profileName, childArgs := extractProfileFlag(args)
		if len(childArgs) > 0 && childArgs[0] == "--" {
			childArgs = childArgs[1:]
		}
		if len(childArgs) == 0 {
			return fmt.Errorf("run: a command to execute is required after --profile")
		}

		secrets, err := loadProfileSecrets(profileName)
		if err != nil {
			return err
		}

		return runChild(childArgs, secrets)
	},
}

func runChild(command []string, secrets map[string]string) error {
	if err := executor.RunCommand(command, secrets); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("run: %w", err)
	}
	return nil
}

// extractProfileFlag scans args for --profile <value> or --profile=<value>.
func extractProfileFlag(args []string) (profileName string, remaining []string) {
	remaining = make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--profile" && i+1 < len(args) {
			profileName = args[i+1]
			i++ // consume the value token
			continue
		}
		if after, found := strings.CutPrefix(args[i], "--profile="); found {
			profileName = after
			continue
		}
		remaining = append(remaining, args[i])
	}
	return
}

func loadProfileSecrets(profileName string) (map[string]string, error) {
	config, err := project.Find()
	if err != nil {
		return nil, fmt.Errorf("run: %w", err)
	}
	profile, err := config.Profile(profileName)
	if err != nil {
		return nil, fmt.Errorf("run: %w", err)
	}
	provider, err := vaults.NewOnePasswordProvider()
	if err != nil {
		return nil, fmt.Errorf("run: %w", err)
	}
	return provider.Fetch(context.Background(), vaults.OnePasswordReference{
		Account: profile.Account,
		Vault:   profile.Vault,
		ItemID:  profile.ItemID,
		Item:    profile.Item,
	})
}

func init() {
	rootCmd.AddCommand(runCmd)
}
