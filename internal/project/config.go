// Package project reads the non-secret inject.toml project configuration.
package project

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

const FileName = "inject.toml"

var identifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type Config struct {
	FormatVersion  int                      `toml:"format_version"`
	ProjectID      string                   `toml:"project_id"`
	Cache          CachePolicy              `toml:"cache"`
	Profiles       map[string]Profile       `toml:"profiles"`
	Commands       map[string]Binding       `toml:"commands"`
	ScriptBindings map[string]ScriptBinding `toml:"script_bindings"`
}

type CachePolicy struct {
	Enabled bool     `toml:"enabled"`
	MaxAge  Duration `toml:"max_age"`
}

type Duration struct {
	time.Duration
	set bool
}

func (duration *Duration) UnmarshalText(text []byte) error {
	parsed, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	duration.Duration = parsed
	duration.set = true
	return nil
}

type Profile struct {
	Provider string `toml:"provider"`
	Account  string `toml:"account"`
	Vault    string `toml:"vault"`
	ItemID   string `toml:"item_id"`
	Item     string `toml:"item"`
}

type Binding struct {
	Profile string   `toml:"profile"`
	Command []string `toml:"command"`
}

type ScriptBinding struct {
	Profile        string `toml:"profile"`
	PackageManager string `toml:"package_manager"`
	Wrapper        string `toml:"wrapper"`
	Script         string `toml:"script"`
	Original       string `toml:"original"`
	PreScript      string `toml:"pre_script"`
	PreOriginal    string `toml:"pre_original"`
	PostScript     string `toml:"post_script"`
	PostOriginal   string `toml:"post_original"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("project: read configuration: %w", err)
	}
	return Parse(data)
}

func Find() (Config, error) {
	return Load(filepath.Join(".", FileName))
}

func Parse(data []byte) (Config, error) {
	var config Config
	metadata, err := toml.NewDecoder(bytes.NewReader(data)).Decode(&config)
	if err != nil {
		return Config{}, fmt.Errorf("project: invalid configuration: %w", err)
	}
	if undecoded := metadata.Undecoded(); len(undecoded) > 0 {
		return Config{}, fmt.Errorf("project: unsupported configuration field %q", undecoded[0])
	}
	if err := config.validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (config Config) Profile(name string) (Profile, error) {
	if name == "" {
		name = "default"
	}
	profile, exists := config.Profiles[name]
	if !exists {
		return Profile{}, fmt.Errorf("project: unknown profile %q", name)
	}
	return profile, nil
}

func (config Config) Binding(name string) (Binding, error) {
	binding, exists := config.Commands[name]
	if !exists {
		return Binding{}, fmt.Errorf("project: unknown command binding %q", name)
	}
	return binding, nil
}

func (config Config) ScriptBinding(name string) (ScriptBinding, error) {
	binding, exists := config.ScriptBindings[name]
	if !exists {
		return ScriptBinding{}, fmt.Errorf("project: unknown package script binding %q", name)
	}
	return binding, nil
}

func (config *Config) validate() error {
	if config.FormatVersion != 1 {
		return fmt.Errorf("project: format_version must be 1")
	}
	if !identifier.MatchString(config.ProjectID) {
		return fmt.Errorf("project: project_id must be a stable identifier")
	}
	if len(config.Profiles) == 0 {
		return fmt.Errorf("project: at least one profile is required")
	}
	if config.Cache.Enabled && !config.Cache.MaxAge.set {
		config.Cache.MaxAge.Duration = 24 * time.Hour
	}
	if config.Cache.MaxAge.set && config.Cache.MaxAge.Duration <= 0 {
		return fmt.Errorf("project: cache max_age must be positive")
	}
	for name, profile := range config.Profiles {
		if !identifier.MatchString(name) {
			return fmt.Errorf("project: invalid profile name %q", name)
		}
		if profile.Provider != "1password" && profile.Provider != "bitwarden" && profile.Provider != "local" {
			return fmt.Errorf("project: profile %q provider must be 1password, bitwarden, or local", name)
		}
		if profile.Provider == "local" {
			continue
		}
		if profile.Provider == "1password" && (strings.TrimSpace(profile.Account) == "" || strings.TrimSpace(profile.Vault) == "") {
			return fmt.Errorf("project: profile %q requires account and vault", name)
		}
		if strings.TrimSpace(profile.ItemID) == "" && strings.TrimSpace(profile.Item) == "" {
			return fmt.Errorf("project: profile %q requires item_id or item", name)
		}
		if strings.Contains(profile.Item, "://") || strings.HasPrefix(profile.Item, "op://") {
			return fmt.Errorf("project: profile %q item must not be a private URL", name)
		}
	}
	for name, binding := range config.Commands {
		if !identifier.MatchString(name) || len(binding.Command) == 0 || strings.TrimSpace(binding.Command[0]) == "" {
			return fmt.Errorf("project: invalid command binding %q", name)
		}
		if _, err := config.Profile(binding.Profile); err != nil {
			return fmt.Errorf("project: command binding %q: %w", name, err)
		}
	}
	for name, binding := range config.ScriptBindings {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(binding.Wrapper) == "" || strings.TrimSpace(binding.Script) == "" {
			return fmt.Errorf("project: invalid package script binding %q", name)
		}
		if binding.PackageManager != "npm" && binding.PackageManager != "pnpm" && binding.PackageManager != "yarn" && binding.PackageManager != "bun" {
			return fmt.Errorf("project: package script binding %q has unsupported package manager", name)
		}
		if _, err := config.Profile(binding.Profile); err != nil {
			return fmt.Errorf("project: package script binding %q: %w", name, err)
		}
	}
	return nil
}
