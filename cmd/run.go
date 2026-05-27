package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/harsh-sonkar/env-pull/internal/crypto"
	"github.com/harsh-sonkar/env-pull/internal/executor"
	"github.com/harsh-sonkar/env-pull/internal/parser"
	"github.com/harsh-sonkar/env-pull/internal/vaults"
)

// runCmd is intentionally thin: it resolves secrets from the appropriate
// backend, then delegates all execution to executor.RunCommand.
var runCmd = &cobra.Command{
	Use:   "run [--aws-secret <name>] <command> [args...]",
	Short: "Run a command with vault secrets injected into its environment",
	Long: `run is the core command of env-pull. It resolves secrets from a vault,
then spawns your target command as a transparent child process with those
secrets present in its environment. The child sees them as ordinary env vars;
they vanish when it exits. Nothing is written to disk.

SECRET SOURCES  (mutually exclusive)

  --aws-secret <name>
    Fetches the named secret from AWS Secrets Manager using your existing
    local credentials — no new tokens or config files required. The secret
    must be stored as a flat JSON object of string pairs:
      {"DB_PASSWORD": "s3cr3t", "API_KEY": "abc123"}
    Credentials are resolved in the standard AWS chain:
      AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY env vars
      → ~/.aws/credentials (shared credentials file)
      → AWS_PROFILE
      → EC2 / ECS / EKS IAM instance role

  (no flag — default)
    Decrypts ` + defaultVaultFile + ` (managed by 'env-pull edit') and
    injects its contents. If the vault file does not exist, the command
    runs with no extra secrets — useful for environments where secrets
    arrive via the ambient environment already.

USAGE EXAMPLES

  # Inject from AWS Secrets Manager
  env-pull run --aws-secret prod/my-app -- ./server

  # Inject from local encrypted vault
  env-pull run -- ./server

  # Pass flags to the child command (use -- to stop env-pull flag parsing)
  env-pull run --aws-secret prod/db -- psql --host=localhost mydb

  # Works with any program, including shell built-ins via a shell wrapper
  env-pull run -- printenv DB_PASSWORD

NOTE: all env-pull flags must appear before the child command. Arguments
after the child binary name are passed through unchanged, including flags
that look like -x or --flag (flag parsing is disabled for this subcommand).`,

	// DisableFlagParsing passes all arguments through to the child command
	// unchanged. Our own --aws-secret flag is extracted manually before dispatch.
	DisableFlagParsing: true,

	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("run: a command to execute is required")
		}

		awsSecretName, childArgs := extractAWSSecretFlag(args)
		if len(childArgs) == 0 {
			return fmt.Errorf("run: a command to execute is required after --aws-secret")
		}

		var (
			secrets map[string]string
			err     error
		)
		if awsSecretName != "" {
			secrets, err = fetchAWSSecrets(awsSecretName)
		} else {
			secrets, err = loadSecrets()
		}
		if err != nil {
			return err
		}

		if err := executor.RunCommand(childArgs, secrets); err != nil {
			// Preserve the child's exact exit code so env-pull is a
			// transparent wrapper from the caller's perspective.
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				os.Exit(exitErr.ExitCode())
			}
			return fmt.Errorf("run: %w", err)
		}
		return nil
	},
}

// extractAWSSecretFlag scans args for --aws-secret <value> or
// --aws-secret=<value>, returns the value and the remaining args with the flag
// and its value removed. This manual parse is required because DisableFlagParsing
// prevents Cobra from processing any flags, including our own.
func extractAWSSecretFlag(args []string) (secretName string, remaining []string) {
	remaining = make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--aws-secret" && i+1 < len(args) {
			secretName = args[i+1]
			i++ // consume the value token
			continue
		}
		if after, found := strings.CutPrefix(args[i], "--aws-secret="); found {
			secretName = after
			continue
		}
		remaining = append(remaining, args[i])
	}
	return
}

// fetchAWSSecrets creates an AWSProvider using the default credential chain and
// fetches the named secret. The returned map is used only for the duration of
// RunCommand and then falls out of scope.
func fetchAWSSecrets(secretName string) (map[string]string, error) {
	ctx := context.Background()
	provider, err := vaults.NewAWSProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("run: %w", err)
	}
	secrets, err := provider.Fetch(ctx, secretName)
	if err != nil {
		return nil, fmt.Errorf("run: %w", err)
	}
	return secrets, nil
}

// loadSecrets decrypts the local vault file and parses it into a key-value map.
// Returns nil (no secrets, no error) when the vault file does not yet exist.
// The encryption key and plaintext are zeroed before return to minimise the
// window during which sensitive bytes are live in memory.
// Note: map values are Go strings and cannot be zeroed without unsafe; let the
// map fall out of scope as soon as RunCommand returns.
func loadSecrets() (map[string]string, error) {
	if _, err := os.Stat(defaultVaultFile); os.IsNotExist(err) {
		return nil, nil
	}

	key, err := crypto.LoadOrCreateKey()
	if err != nil {
		return nil, fmt.Errorf("run: %w", err)
	}
	defer func() {
		for i := range key {
			key[i] = 0
		}
	}()

	data, err := os.ReadFile(defaultVaultFile)
	if err != nil {
		return nil, fmt.Errorf("run: failed to read vault: %w", err)
	}

	plaintext, err := crypto.Decrypt(data, key)
	if err != nil {
		return nil, fmt.Errorf("run: failed to decrypt vault: %w", err)
	}
	defer func() {
		for i := range plaintext {
			plaintext[i] = 0
		}
	}()

	secrets, err := parser.Parse(plaintext)
	if err != nil {
		return nil, fmt.Errorf("run: failed to parse vault: %w", err)
	}
	return secrets, nil
}

func init() {
	rootCmd.AddCommand(runCmd)
}
