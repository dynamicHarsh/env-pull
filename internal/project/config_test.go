package project_test

import (
	"strings"
	"testing"

	"github.com/harsh-sonkar/env-pull/internal/project"
)

func TestParse(t *testing.T) {
	valid := []byte(`format_version = 1
project_id = "billing-api"

[profiles.default]
provider = "1password"
account = "acme.1password.com"
vault = "Engineering"
item_id = "abc123"

[commands.dev]
profile = "default"
command = ["go", "run", "."]
`)

	config, err := project.Parse(valid)
	if err != nil {
		t.Fatalf("Parse(valid) error = %v", err)
	}
	if config.ProjectID != "billing-api" {
		t.Errorf("ProjectID = %q, want billing-api", config.ProjectID)
	}
	if got := config.Commands["dev"].Command; strings.Join(got, " ") != "go run ." {
		t.Errorf("dev command = %q, want go run .", got)
	}
}

func TestParseRejectsUnsafeOrAmbiguousConfiguration(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "unknown provider",
			input: `format_version = 1
project_id = "billing-api"
[profiles.default]
provider = "aws"
account = "acme"
vault = "Engineering"
item_id = "abc123"`,
		},
		{
			name: "secret value",
			input: `format_version = 1
project_id = "billing-api"
token = "secret"
[profiles.default]
provider = "1password"
account = "acme"
vault = "Engineering"
item_id = "abc123"`,
		},
		{
			name: "unknown binding profile",
			input: `format_version = 1
project_id = "billing-api"
[profiles.default]
provider = "1password"
account = "acme"
vault = "Engineering"
item_id = "abc123"
[commands.dev]
profile = "missing"
command = ["go", "run", "."]`,
		},
		{
			name: "private item URL",
			input: `format_version = 1
project_id = "billing-api"
[profiles.default]
provider = "1password"
account = "acme"
vault = "Engineering"
item = "https://start.1password.com/open/i?a=private"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := project.Parse([]byte(test.input)); err == nil {
				t.Fatal("Parse() error = nil, want invalid configuration rejected")
			}
		})
	}
}
