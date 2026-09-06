package project_test

import (
	"strings"
	"testing"
	"time"

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

func TestParseAcceptsBitwardenProfile(t *testing.T) {
	config, err := project.Parse([]byte(`format_version = 1
project_id = "billing-api"

[profiles.default]
provider = "bitwarden"
item_id = "abc123"
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if config.Profiles["default"].Provider != "bitwarden" {
		t.Errorf("provider = %q, want bitwarden", config.Profiles["default"].Provider)
	}
}

func TestParseAcceptsPackageScriptOwnershipMetadata(t *testing.T) {
	config, err := project.Parse([]byte(`format_version = 1
project_id = "billing-api"

[profiles.default]
provider = "local"

[script_bindings."dev:web"]
profile = "default"
package_manager = "npm"
wrapper = "inject __run-package-script \"dev\""
script = "inject:original:dev"
original = "vite --host 0.0.0.0 | tee app.log"
pre_script = "inject:original:predev"
pre_original = "printf 'ready\\n'"
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	binding, err := config.ScriptBinding("dev:web")
	if err != nil {
		t.Fatalf("ScriptBinding() error = %v", err)
	}
	if binding.Original != "vite --host 0.0.0.0 | tee app.log" || binding.PreOriginal != `printf 'ready\n'` {
		t.Errorf("script binding = %#v, want exact restoration values", binding)
	}
}

func TestParseAcceptsOptInRemoteCachePolicy(t *testing.T) {
	config, err := project.Parse([]byte(`format_version = 1
project_id = "billing-api"

[cache]
enabled = true

[profiles.default]
provider = "bitwarden"
item_id = "abc123"
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !config.Cache.Enabled {
		t.Error("cache is not enabled")
	}
	if config.Cache.MaxAge.Duration != 24*time.Hour {
		t.Errorf("cache maximum age = %v, want 24h", config.Cache.MaxAge.Duration)
	}
}

func TestParseRejectsNonPositiveCacheMaximumAge(t *testing.T) {
	_, err := project.Parse([]byte(`format_version = 1
project_id = "billing-api"

[cache]
enabled = true
max_age = "0s"

[profiles.default]
provider = "bitwarden"
item_id = "abc123"
`))
	if err == nil {
		t.Fatal("Parse() error = nil, want non-positive cache maximum age rejected")
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
