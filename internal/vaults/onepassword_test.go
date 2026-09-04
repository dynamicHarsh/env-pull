package vaults

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type fakeCommandRunner struct {
	args   []string
	output []byte
	err    error
}

func (runner *fakeCommandRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	runner.args = args
	return runner.output, runner.err
}

func TestOnePasswordProviderFetch(t *testing.T) {
	t.Run("fetches secure note by immutable item ID", func(t *testing.T) {
		runner := &fakeCommandRunner{output: []byte(`{"fields":[{"id":"notesPlain","value":"TOKEN=from-1password\n"}]}`)}
		provider := newOnePasswordProviderWithRunner(runner)

		secrets, err := provider.Fetch(context.Background(), OnePasswordReference{
			Account: "acme.1password.com",
			Vault:   "Engineering",
			ItemID:  "immutable-id",
			Item:    "ignored-display-name",
		})
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}
		if secrets["TOKEN"] != "from-1password" {
			t.Errorf("TOKEN = %q, want from-1password", secrets["TOKEN"])
		}
		wantArgs := []string{"item", "get", "immutable-id", "--vault", "Engineering", "--account", "acme.1password.com", "--format", "json"}
		if !reflect.DeepEqual(runner.args, wantArgs) {
			t.Errorf("op args = %q, want %q", runner.args, wantArgs)
		}
	})

	t.Run("provider failure is redacted", func(t *testing.T) {
		runner := &fakeCommandRunner{err: errors.New("not signed in: TOKEN=secret")}
		provider := newOnePasswordProviderWithRunner(runner)

		_, err := provider.Fetch(context.Background(), OnePasswordReference{Account: "acme", Vault: "Engineering", ItemID: "id"})
		if err == nil {
			t.Fatal("Fetch() error = nil, want provider failure")
		}
		if got := err.Error(); got == "" || containsSecret(got) {
			t.Errorf("Fetch() error = %q, want redacted remediation", got)
		}
	})
}

func containsSecret(message string) bool { return strings.Contains(message, "TOKEN=secret") }
