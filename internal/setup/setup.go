// Package setup guides a project from a legacy .env workflow to inject.
package setup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/harsh-sonkar/env-pull/internal/executor"
	"github.com/harsh-sonkar/env-pull/internal/parser"
	"github.com/harsh-sonkar/env-pull/internal/project"
	"github.com/harsh-sonkar/env-pull/internal/store"
	"github.com/harsh-sonkar/env-pull/internal/vaults"
)

var ErrOnePasswordUnavailable = errors.New("1password unavailable")

type Request struct {
	Directory        string
	ProjectID        string
	Provider         string
	Account          string
	Vault            string
	ItemID           string
	Item             string
	Binding          string
	Command          []string
	PackageScript    string
	Validate         []string
	Local            bool
	Confirm          bool
	RemoveLegacyEnv  bool
	ConfirmRemoveEnv bool
	CheckOnePassword func() error
	RunValidation    func([]string) error
	Store            store.Store
	Output           io.Writer
}

// Run previews and, after confirmation, applies a non-secret inject configuration.
func Run(request Request) error {
	if request.Directory == "" {
		request.Directory = "."
	}
	if request.Output == nil {
		request.Output = io.Discard
	}
	legacyEnvPath := filepath.Join(request.Directory, ".env")
	_, legacyEnvExists := statFile(legacyEnvPath)
	selectSource(&request, legacyEnvExists)
	if request.ProjectID == "" {
		projectID, err := defaultProjectID(request.Directory)
		if err != nil {
			return err
		}
		request.ProjectID = projectID
	}
	if err := validateRequest(request); err != nil {
		return err
	}
	if !request.Local {
		if err := checkOnePassword(request); err != nil {
			fmt.Fprintln(request.Output, "1Password is unavailable; run `op signin` and retry")
			return err
		}
	}
	if _, exists := statFile(filepath.Join(request.Directory, project.FileName)); exists {
		return fmt.Errorf("setup: inject.toml already exists")
	}

	if legacyEnvExists {
		fmt.Fprintln(request.Output, "Detected legacy .env")
	}
	for _, name := range candidatePackageScripts(request.Directory) {
		fmt.Fprintf(request.Output, "Candidate command: %s\n", name)
	}

	config, packageChange, err := plan(request)
	if err != nil {
		return err
	}
	configData, err := encodeConfig(config)
	if err != nil {
		return err
	}
	fmt.Fprintln(request.Output, "Will write inject.toml:")
	fmt.Fprint(request.Output, string(configData))
	if packageChange != nil {
		fmt.Fprintf(request.Output, "Will update package.json script %q to run inject %s\n", request.PackageScript, request.PackageScript)
	}
	if !request.Confirm {
		fmt.Fprintln(request.Output, "No changes made; rerun with explicit confirmation")
		return nil
	}
	if !request.Local || len(request.Validate) > 0 {
		if err := validateValidationCommand(request.Validate); err != nil {
			return err
		}
	}
	var localSecrets map[string]string
	if request.Local {
		if request.Store == nil {
			return fmt.Errorf("setup: local credential store is unavailable")
		}
		data, err := os.ReadFile(legacyEnvPath)
		if err != nil {
			return fmt.Errorf("setup: read legacy .env: %w", err)
		}
		localSecrets, err = parser.Parse(data)
		if err != nil {
			return fmt.Errorf("setup: import legacy .env: %w", err)
		}
		if len(request.Validate) > 0 {
			if err := runValidation(request, localSecrets); err != nil {
				return fmt.Errorf("setup: validation failed: %w", err)
			}
		}
		if err := request.Store.Put(request.ProjectID, "default", localSecrets); err != nil {
			return fmt.Errorf("setup: save local secret set: %w", err)
		}
	}

	if err := os.WriteFile(filepath.Join(request.Directory, project.FileName), configData, 0o644); err != nil {
		return fmt.Errorf("setup: write inject.toml: %w", err)
	}
	if packageChange != nil {
		if err := os.WriteFile(filepath.Join(request.Directory, "package.json"), packageChange, 0o644); err != nil {
			return fmt.Errorf("setup: update package.json: %w", err)
		}
	}
	if !request.Local {
		if err := runValidation(request, localSecrets); err != nil {
			return fmt.Errorf("setup: validation failed: %w", err)
		}
	}
	fmt.Fprintln(request.Output, "Validation succeeded")
	if request.RemoveLegacyEnv && legacyEnvExists {
		if !request.ConfirmRemoveEnv {
			fmt.Fprintln(request.Output, "Legacy .env was retained; its removal requires explicit confirmation")
			return nil
		}
		if err := os.Remove(legacyEnvPath); err != nil {
			return fmt.Errorf("setup: remove legacy .env: %w", err)
		}
		fmt.Fprintln(request.Output, "Removed legacy .env")
	}
	return nil
}

func selectSource(request *Request, legacyEnvExists bool) {
	if request.Local || request.Provider != "" || hasRemoteReference(*request) || !legacyEnvExists {
		return
	}
	request.Local = true
}

func hasRemoteReference(request Request) bool {
	return request.Account != "" || request.Vault != "" || request.ItemID != "" || request.Item != ""
}

func defaultProjectID(directory string) (string, error) {
	absoluteDirectory, err := filepath.Abs(directory)
	if err != nil {
		return "", fmt.Errorf("setup: determine project directory: %w", err)
	}
	return filepath.Base(absoluteDirectory), nil
}

func validateRequest(request Request) error {
	if strings.TrimSpace(request.ProjectID) == "" {
		return fmt.Errorf("setup: project ID is required")
	}
	if request.Provider != "" && request.Provider != "1password" {
		return fmt.Errorf("setup: unsupported provider %q", request.Provider)
	}
	if request.Provider == "1password" && request.Local {
		return fmt.Errorf("setup: provider and local source cannot be selected together")
	}
	if request.Local {
		return nil
	}
	if strings.TrimSpace(request.Account) == "" || strings.TrimSpace(request.Vault) == "" {
		return fmt.Errorf("setup: project ID, 1Password account, and vault are required")
	}
	if strings.TrimSpace(request.ItemID) == "" && strings.TrimSpace(request.Item) == "" {
		return fmt.Errorf("setup: a 1Password item ID or item name is required")
	}
	if request.PackageScript != "" && request.Binding != "" && request.Binding != request.PackageScript {
		return fmt.Errorf("setup: package script and binding must have the same name")
	}
	if request.Binding != "" && len(request.Command) == 0 && request.PackageScript == "" {
		return fmt.Errorf("setup: binding %q requires a command", request.Binding)
	}
	return nil
}

func checkOnePassword(request Request) error {
	if request.CheckOnePassword != nil {
		return request.CheckOnePassword()
	}
	path, err := exec.LookPath("op")
	if err != nil {
		return ErrOnePasswordUnavailable
	}
	if err := exec.Command(path, "account", "list").Run(); err != nil {
		return ErrOnePasswordUnavailable
	}
	return nil
}

func plan(request Request) (project.Config, []byte, error) {
	profile := project.Profile{
		Provider: "1password", Account: request.Account, Vault: request.Vault, ItemID: request.ItemID, Item: request.Item,
	}
	if request.Local {
		profile = project.Profile{Provider: "local"}
	}
	config := project.Config{
		FormatVersion: 1,
		ProjectID:     request.ProjectID,
		Profiles:      map[string]project.Profile{"default": profile},
		Commands:      map[string]project.Binding{},
	}
	if request.PackageScript == "" && request.Binding == "" {
		return config, nil, nil
	}
	binding := request.Binding
	if binding == "" {
		binding = request.PackageScript
	}
	command := request.Command
	var packageChange []byte
	if request.PackageScript != "" {
		var err error
		command, packageChange, err = packageBinding(request.Directory, request.PackageScript)
		if err != nil {
			return project.Config{}, nil, err
		}
		if command == nil {
			fmt.Fprintf(request.Output, "Script %q is not safe to rewrite; creating an explicit binding instead\n", request.PackageScript)
			command = request.Command
			packageChange = nil
		}
	}
	if len(command) == 0 {
		return project.Config{}, nil, fmt.Errorf("setup: binding %q requires a command", binding)
	}
	config.Commands[binding] = project.Binding{Command: command}
	return config, packageChange, nil
}

func encodeConfig(config project.Config) ([]byte, error) {
	profile := config.Profiles["default"]
	data := []byte(fmt.Sprintf("format_version = 1\nproject_id = %q\n\n[profiles.default]\nprovider = %q\n", config.ProjectID, profile.Provider))
	if profile.Provider == "1password" {
		data = append(data, fmt.Sprintf("account = %q\nvault = %q\n", profile.Account, profile.Vault)...)
	}
	if profile.Provider == "local" {
		goto bindings
	}
	if profile.ItemID != "" {
		data = append(data, fmt.Sprintf("item_id = %q\n", profile.ItemID)...)
	} else {
		data = append(data, fmt.Sprintf("item = %q\n", profile.Item)...)
	}
bindings:
	for name, binding := range config.Commands {
		command, err := json.Marshal(binding.Command)
		if err != nil {
			return nil, err
		}
		data = append(data, fmt.Sprintf("\n[commands.%s]\nprofile = \"default\"\ncommand = %s\n", name, command)...)
	}
	if _, err := project.Parse(data); err != nil {
		return nil, fmt.Errorf("setup: invalid planned configuration: %w", err)
	}
	return data, nil
}

func packageBinding(directory, script string) ([]string, []byte, error) {
	path := filepath.Join(directory, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("setup: read package.json: %w", err)
	}
	var manifest map[string]json.RawMessage
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, nil, fmt.Errorf("setup: invalid package.json: %w", err)
	}
	var scripts map[string]string
	if err := json.Unmarshal(manifest["scripts"], &scripts); err != nil {
		return nil, nil, fmt.Errorf("setup: package.json scripts must be an object: %w", err)
	}
	value, exists := scripts[script]
	if !exists {
		return nil, nil, fmt.Errorf("setup: package.json has no %q script", script)
	}
	command, safe := safeCommand(value)
	if !safe {
		return nil, nil, nil
	}
	scripts[script] = "inject " + script
	encodedScripts, err := json.Marshal(scripts)
	if err != nil {
		return nil, nil, err
	}
	manifest["scripts"] = encodedScripts
	updated, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return command, append(updated, '\n'), nil
}

func candidatePackageScripts(directory string) []string {
	data, err := os.ReadFile(filepath.Join(directory, "package.json"))
	if err != nil {
		return nil
	}
	var manifest struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(data, &manifest) != nil {
		return nil
	}
	candidates := make([]string, 0, len(manifest.Scripts))
	for name := range manifest.Scripts {
		candidates = append(candidates, name)
	}
	sort.Strings(candidates)
	return candidates
}

func safeCommand(value string) ([]string, bool) {
	if strings.TrimSpace(value) == "" || strings.ContainsAny(value, "|&;<>()`$\\\n\r\"'") {
		return nil, false
	}
	parts := strings.Fields(value)
	for _, part := range parts {
		if strings.IndexFunc(part, unicode.IsSpace) >= 0 {
			return nil, false
		}
	}
	return parts, len(parts) > 0
}

func runValidation(request Request, localSecrets map[string]string) error {
	if request.RunValidation != nil {
		return request.RunValidation(request.Validate)
	}
	if request.Local {
		return executor.RunCommand(request.Validate, localSecrets)
	}
	provider, err := vaults.NewOnePasswordProvider()
	if err != nil {
		return err
	}
	secrets, err := provider.Fetch(requestContext(), vaults.OnePasswordReference{Account: request.Account, Vault: request.Vault, ItemID: request.ItemID, Item: request.Item})
	if err != nil {
		return err
	}
	return executor.RunCommand(request.Validate, secrets)
}

func requestContext() context.Context { return context.Background() }

func validateValidationCommand(command []string) error {
	if len(command) == 0 {
		return fmt.Errorf("setup: a finite validation command is required")
	}
	if containsLongRunningMarker(command) {
		return fmt.Errorf("setup: validation command must be finite")
	}
	return nil
}

func containsLongRunningMarker(command []string) bool {
	for _, argument := range command {
		switch strings.ToLower(argument) {
		case "dev", "serve", "server", "start", "watch":
			return true
		}
	}
	return false
}

func statFile(path string) (os.FileInfo, bool) {
	info, err := os.Stat(path)
	return info, err == nil && !info.IsDir()
}
