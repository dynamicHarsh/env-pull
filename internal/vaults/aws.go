package vaults

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// secretsManagerClient is the minimal API surface consumed from
// secretsmanager.Client. Declaring it as an interface allows a lightweight mock
// to be injected in tests without any network calls or AWS credentials.
type secretsManagerClient interface {
	GetSecretValue(
		ctx context.Context,
		params *secretsmanager.GetSecretValueInput,
		optFns ...func(*secretsmanager.Options),
	) (*secretsmanager.GetSecretValueOutput, error)
}

// AWSProvider fetches secrets from AWS Secrets Manager. It uses the developer's
// existing local AWS auth context (AWS_PROFILE, ~/.aws/credentials, EC2/ECS
// instance role, etc.) — no additional credentials need to be configured.
type AWSProvider struct {
	client secretsManagerClient
}

// NewAWSProvider creates an AWSProvider by loading the default AWS credential
// chain. This is zero-config: it reuses whatever the developer already has set
// up locally (shared credentials file, environment variables, SSO, IAM role).
func NewAWSProvider(ctx context.Context) (*AWSProvider, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"aws: failed to load AWS configuration — ensure ~/.aws/credentials, "+
				"AWS_PROFILE, or an IAM instance role is available: %w", err)
	}
	return &AWSProvider{client: secretsmanager.NewFromConfig(cfg)}, nil
}

// newAWSProviderWithClient creates an AWSProvider backed by a custom client.
// Used only in tests to inject a mock without network I/O.
func newAWSProviderWithClient(c secretsManagerClient) *AWSProvider {
	return &AWSProvider{client: c}
}

// Fetch retrieves secretName from AWS Secrets Manager and parses its JSON
// string payload into a map[string]string. The secret must be stored as a JSON
// object whose values are all strings (e.g. {"DB_PASS":"s3cr3t","PORT":"5432"}).
//
// Security: the raw secret bytes are zeroed as soon as JSON parsing completes.
// The *string field on the SDK response is nil-ed to allow GC to reclaim it;
// the underlying Go string value cannot be zeroed without unsafe.
func (p *AWSProvider) Fetch(ctx context.Context, secretName string) (map[string]string, error) {
	out, err := p.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &secretName,
	})
	if err != nil {
		return nil, fmt.Errorf(
			"aws: GetSecretValue(%q) failed — verify the secret exists, the AWS "+
				"region is correct, and the caller has secretsmanager:GetSecretValue "+
				"permission: %w", secretName, err)
	}

	if out.SecretString == nil {
		return nil, fmt.Errorf(
			"aws: secret %q contains binary data; only JSON string-value secrets "+
				"are supported — store secrets as a JSON object of string pairs", secretName)
	}

	// Copy into a byte slice so we can zero the buffer after parsing.
	// Nil-ing the SDK pointer releases the reference; the Go runtime may GC it.
	raw := []byte(*out.SecretString)
	out.SecretString = nil
	defer func() {
		for i := range raw {
			raw[i] = 0
		}
	}()

	var kv map[string]string
	if err := json.Unmarshal(raw, &kv); err != nil {
		return nil, fmt.Errorf(
			"aws: secret %q payload is not a valid JSON string-to-string object "+
				"(e.g. {\"KEY\":\"value\"}): %w", secretName, err)
	}
	return kv, nil
}
