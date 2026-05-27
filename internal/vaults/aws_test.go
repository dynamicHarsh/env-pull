// Uses package vaults (not vaults_test) to access the unexported
// secretsManagerClient interface and newAWSProviderWithClient constructor.
package vaults

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// mockSMClient implements secretsManagerClient for testing without network I/O.
type mockSMClient struct {
	output *secretsmanager.GetSecretValueOutput
	err    error
}

func (m *mockSMClient) GetSecretValue(
	_ context.Context,
	_ *secretsmanager.GetSecretValueInput,
	_ ...func(*secretsmanager.Options),
) (*secretsmanager.GetSecretValueOutput, error) {
	return m.output, m.err
}

func secretOutput(s string) *secretsmanager.GetSecretValueOutput {
	return &secretsmanager.GetSecretValueOutput{SecretString: &s}
}

func TestAWSProviderFetch(t *testing.T) {
	tests := []struct {
		name      string
		output    *secretsmanager.GetSecretValueOutput
		clientErr error
		wantMap   map[string]string
		wantErr   bool
	}{
		{
			name:    "valid JSON object with multiple pairs",
			output:  secretOutput(`{"DB_PASSWORD":"s3cr3t","API_KEY":"abc123"}`),
			wantMap: map[string]string{"DB_PASSWORD": "s3cr3t", "API_KEY": "abc123"},
		},
		{
			name:    "valid JSON object with single pair",
			output:  secretOutput(`{"TOKEN":"xyz789"}`),
			wantMap: map[string]string{"TOKEN": "xyz789"},
		},
		{
			name:    "empty JSON object returns empty map",
			output:  secretOutput(`{}`),
			wantMap: map[string]string{},
		},
		{
			name:    "invalid JSON returns descriptive error",
			output:  secretOutput(`not-json`),
			wantErr: true,
		},
		{
			name:    "JSON array (not object) returns error",
			output:  secretOutput(`["a","b"]`),
			wantErr: true,
		},
		{
			name:    "JSON object with non-string value returns error",
			output:  secretOutput(`{"KEY":123}`),
			wantErr: true,
		},
		{
			name:    "nil SecretString (binary secret) returns descriptive error",
			output:  &secretsmanager.GetSecretValueOutput{SecretString: nil},
			wantErr: true,
		},
		{
			name:      "AWS API error is wrapped with context",
			clientErr: errors.New("ResourceNotFoundException: secret not found"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newAWSProviderWithClient(&mockSMClient{output: tt.output, err: tt.clientErr})

			got, err := p.Fetch(context.Background(), "test/my-secret")

			if (err != nil) != tt.wantErr {
				t.Fatalf("Fetch() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.wantMap) {
				t.Fatalf("Fetch() returned %d entries, want %d; got = %v", len(got), len(tt.wantMap), got)
			}
			for k, wantV := range tt.wantMap {
				gotV, ok := got[k]
				if !ok {
					t.Errorf("Fetch() missing key %q", k)
					continue
				}
				if gotV != wantV {
					t.Errorf("Fetch()[%q] = %q, want %q", k, gotV, wantV)
				}
			}
		})
	}
}
