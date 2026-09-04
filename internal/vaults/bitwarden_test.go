package vaults

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestBitwardenProviderFetch(t *testing.T) {
	t.Run("fetches and parses a secure note by immutable item ID", func(t *testing.T) {
		runner := &fakeCommandRunner{output: []byte("TOKEN=from-bitwarden\n")}
		provider := newBitwardenProviderWithRunner(runner)

		secrets, err := provider.Fetch(context.Background(), BitwardenReference{
			ItemID: "immutable-id",
			Item:   "ignored-display-name",
		})
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}
		if secrets["TOKEN"] != "from-bitwarden" {
			t.Errorf("TOKEN = %q, want from-bitwarden", secrets["TOKEN"])
		}
		wantArgs := []string{"get", "notes", "immutable-id"}
		if !reflect.DeepEqual(runner.args, wantArgs) {
			t.Errorf("bw args = %q, want %q", runner.args, wantArgs)
		}
	})

	t.Run("provider failure is redacted", func(t *testing.T) {
		runner := &fakeCommandRunner{err: errors.New("not logged in: TOKEN=secret")}
		provider := newBitwardenProviderWithRunner(runner)

		_, err := provider.Fetch(context.Background(), BitwardenReference{ItemID: "id"})
		if err == nil {
			t.Fatal("Fetch() error = nil, want provider failure")
		}
		if got := err.Error(); got == "" || strings.Contains(got, "TOKEN=secret") {
			t.Errorf("Fetch() error = %q, want redacted remediation", got)
		}
	})
}

func TestRemoteProvidersShareStrictSecretSetRules(t *testing.T) {
	tests := []struct {
		name    string
		onePass []byte
		bitward []byte
		wantErr bool
	}{
		{
			name:    "valid secret set",
			onePass: []byte(`{"fields":[{"id":"notesPlain","value":"TOKEN=same-value\n"}]}`),
			bitward: []byte("TOKEN=same-value\n"),
		},
		{
			name:    "shell expansion is rejected",
			onePass: []byte(`{"fields":[{"id":"notesPlain","value":"TOKEN=${OTHER}\n"}]}`),
			bitward: []byte("TOKEN=${OTHER}\n"),
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			onePassword, onePasswordErr := newOnePasswordProviderWithRunner(&fakeCommandRunner{output: test.onePass}).Fetch(context.Background(), OnePasswordReference{ItemID: "id"})
			bitwarden, bitwardenErr := newBitwardenProviderWithRunner(&fakeCommandRunner{output: test.bitward}).Fetch(context.Background(), BitwardenReference{ItemID: "id"})

			if (onePasswordErr != nil) != test.wantErr || (bitwardenErr != nil) != test.wantErr {
				t.Fatalf("Fetch() errors = 1password: %v, bitwarden: %v, wantErr %t", onePasswordErr, bitwardenErr, test.wantErr)
			}
			if !test.wantErr && !reflect.DeepEqual(onePassword, bitwarden) {
				t.Errorf("secret sets differ: 1password %q, bitwarden %q", onePassword, bitwarden)
			}
		})
	}
}
