package executor_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/harsh-sonkar/env-pull/internal/executor"
)

// TestMain lets this test binary double as a cross-platform subprocess helper.
// When ENV_PULL_PRINT_VAR is set, it prints the value of that env var to stdout
// and exits immediately — no tests run. This eliminates reliance on platform
// shell builtins (printenv, cmd /C set) for secret-injection verification.
func TestMain(m *testing.M) {
	if key := os.Getenv("ENV_PULL_PRINT_VAR"); key != "" {
		fmt.Println(os.Getenv(key))
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestRunCommand(t *testing.T) {
	// selfBin is the current test binary. When re-invoked as a child process
	// with ENV_PULL_PRINT_VAR set, TestMain intercepts and acts as a helper.
	selfBin, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}

	tests := []struct {
		name     string
		command  []string
		envSetup map[string]string // vars set in the parent process via t.Setenv
		secrets  map[string]string // secrets passed to RunCommand
		wantOut  string            // expected trimmed stdout (empty means skip check)
		wantErr  bool
	}{
		{
			name:     "injects a single secret into the child environment",
			command:  []string{selfBin},
			envSetup: map[string]string{"ENV_PULL_PRINT_VAR": "TEST_SECRET"},
			secrets:  map[string]string{"TEST_SECRET": "hello_from_vault"},
			wantOut:  "hello_from_vault",
		},
		{
			name:     "injects multiple secrets and reads the correct one",
			command:  []string{selfBin},
			envSetup: map[string]string{"ENV_PULL_PRINT_VAR": "DB_PASSWORD"},
			secrets: map[string]string{
				"DB_PASSWORD": "s3cr3t_db",
				"API_KEY":     "abc123",
			},
			wantOut: "s3cr3t_db",
		},
		{
			name:    "returns error for a non-existent command",
			command: []string{"__nonexistent_cmd_8675309__"},
			wantErr: true,
		},
		{
			name:    "returns error for an empty command slice",
			command: []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// t.Setenv propagates into os.Environ(), which RunCommand reads.
			for k, v := range tt.envSetup {
				t.Setenv(k, v)
			}

			// Swap os.Stdout for a pipe so the child's output is capturable.
			// RunCommand does cmd.Stdout = os.Stdout at call time, so the swap
			// must happen before RunCommand is invoked.
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("os.Pipe: %v", err)
			}
			origOut := os.Stdout
			os.Stdout = w

			var buf bytes.Buffer
			copyDone := make(chan struct{})
			go func() {
				io.Copy(&buf, r) //nolint:errcheck
				close(copyDone)
			}()

			runErr := executor.RunCommand(tt.command, tt.secrets)

			w.Close()
			os.Stdout = origOut
			<-copyDone
			r.Close()

			if (runErr != nil) != tt.wantErr {
				t.Errorf("RunCommand() error = %v, wantErr = %v", runErr, tt.wantErr)
			}
			if tt.wantOut != "" {
				got := strings.TrimSpace(buf.String())
				if got != tt.wantOut {
					t.Errorf("stdout = %q, want %q", got, tt.wantOut)
				}
			}
		})
	}
}
