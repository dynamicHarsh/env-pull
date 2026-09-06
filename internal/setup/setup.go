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
	"regexp"
	"sort"
	"strings"

	"github.com/harsh-sonkar/env-pull/internal/executor"
	"github.com/harsh-sonkar/env-pull/internal/parser"
	"github.com/harsh-sonkar/env-pull/internal/project"
	"github.com/harsh-sonkar/env-pull/internal/store"
	"github.com/harsh-sonkar/env-pull/internal/vaults"
)

var ErrOnePasswordUnavailable = errors.New("1password unavailable")

var tomlBareKey = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type Request struct {
	Directory           string
	ProjectID           string
	Provider            string
	Account             string
	Vault               string
	ItemID              string
	Item                string
	Binding             string
	Command             []string
	PackageScript       string
	SelectPackageScript func([]string, string) (string, error)
	Validate            []string
	Local               bool
	SelectedInputs      []string
	Confirm             bool
	RemoveLegacyEnv     bool
	ConfirmRemoveEnv    bool
	CheckOnePassword    func() error
	RunValidation       func([]string) error
	Store               store.Store
	Output              io.Writer
}

type Plan struct {
	ProjectID            string
	Source               string
	PlaintextInputs      []PlaintextInput
	DeveloperCommands    []DeveloperCommand
	ValidationCandidates []string
	ValidationCommand    []string
	PackageManager       string
	FileChanges          []string
	ConfigData           []byte
}

type PlaintextInput struct {
	Name       string
	Profile    string
	Selected   bool
	Precedence string
	Variables  []string
}

type DeveloperCommand struct {
	Name    string
	Default bool
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
	inputs := detectPlaintextInputs(request.Directory)
	if err := selectPlaintextInputs(inputs, request.SelectedInputs); err != nil {
		return err
	}
	selectSource(&request, len(inputs) > 0)
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
	if _, exists := statFile(filepath.Join(request.Directory, project.FileName)); exists {
		return fmt.Errorf("setup: inject.toml already exists")
	}

	candidates := candidatePackageScripts(request.Directory)
	if request.PackageScript == "" && request.Binding == "" && request.SelectPackageScript != nil {
		selection, err := request.SelectPackageScript(candidates, defaultPackageScript(candidates))
		if err != nil {
			return err
		}
		request.PackageScript = selection
	}

	var localProfiles map[string]map[string]string
	if request.Local {
		var err error
		localProfiles, err = composeLocalProfiles(request.Directory, inputs)
		if err != nil {
			return err
		}
	}
	config, err := plan(request, localProfiles)
	if err != nil {
		return err
	}
	configData, err := encodeConfig(config)
	if err != nil {
		return err
	}
	setupPlan := buildPlan(request, inputs, candidates, configData)
	previewPlan(request.Output, setupPlan)
	if !request.Confirm {
		fmt.Fprintln(request.Output, "No changes made; rerun with explicit confirmation")
		return nil
	}
	if !request.Local && effectiveProvider(request) == "1password" {
		if err := checkOnePassword(request); err != nil {
			fmt.Fprintln(request.Output, "1Password is unavailable; run `op signin` and retry")
			return err
		}
	}
	if !request.Local || request.RemoveLegacyEnv || len(request.Validate) > 0 {
		if err := validateValidationCommand(request.Validate); err != nil {
			return err
		}
	}
	if request.Local {
		if request.Store == nil {
			return fmt.Errorf("setup: local credential store is unavailable")
		}
		if len(localProfiles) == 0 {
			return fmt.Errorf("setup: select at least one plaintext input for local setup")
		}
		if len(request.Validate) > 0 {
			if err := runValidation(request, localProfiles["default"]); err != nil {
				return fmt.Errorf("setup: validation failed: %w", err)
			}
		}
		for profile, secrets := range localProfiles {
			if err := request.Store.Put(request.ProjectID, profile, secrets); err != nil {
				return fmt.Errorf("setup: save local profile %q: %w", profile, err)
			}
		}
	}

	if err := os.WriteFile(filepath.Join(request.Directory, project.FileName), configData, 0o644); err != nil {
		return fmt.Errorf("setup: write inject.toml: %w", err)
	}
	if !request.Local {
		if err := runValidation(request, nil); err != nil {
			return fmt.Errorf("setup: validation failed: %w", err)
		}
	}
	fmt.Fprintln(request.Output, "Validation succeeded")
	if request.RemoveLegacyEnv && hasPlaintextInput(inputs, ".env") {
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

func selectSource(request *Request, plaintextInputExists bool) {
	if request.Local || request.Provider != "" || hasRemoteReference(*request) || !plaintextInputExists {
		return
	}
	request.Local = true
}

func buildPlan(request Request, inputs []PlaintextInput, scripts []string, configData []byte) Plan {
	plan := Plan{
		ProjectID:         request.ProjectID,
		Source:            effectiveProvider(request),
		PlaintextInputs:   inputs,
		ValidationCommand: append([]string(nil), request.Validate...),
		PackageManager:    detectPackageManager(request.Directory),
		FileChanges:       []string{"create inject.toml"},
		ConfigData:        configData,
	}
	if request.Local {
		plan.Source = "local"
	}
	for _, name := range scripts {
		plan.DeveloperCommands = append(plan.DeveloperCommands, DeveloperCommand{Name: name, Default: isRuntimeScript(name)})
		if isFiniteValidationScript(name) {
			plan.ValidationCandidates = append(plan.ValidationCandidates, name)
		}
	}
	if request.RemoveLegacyEnv {
		plan.FileChanges = append(plan.FileChanges, "remove .env after successful validation")
	}
	return plan
}

func previewPlan(output io.Writer, plan Plan) {
	fmt.Fprintln(output, "Setup plan:")
	fmt.Fprintf(output, "Project ID: %s\n", plan.ProjectID)
	switch plan.Source {
	case "local":
		fmt.Fprintln(output, "Source choices: Local (selected), 1Password, Bitwarden")
	case "bitwarden":
		fmt.Fprintln(output, "Source choices: Local, 1Password, Bitwarden (selected)")
	default:
		fmt.Fprintln(output, "Source choices: Local, 1Password (selected), Bitwarden")
	}
	for _, input := range plan.PlaintextInputs {
		state := "available"
		if input.Selected {
			state = "selected"
		}
		details := fmt.Sprintf("%s; profile %s", state, input.Profile)
		if input.Precedence != "" {
			details += "; precedence " + input.Precedence
		}
		fmt.Fprintf(output, "Plaintext input: %s (%s)\n", input.Name, details)
		if input.Selected {
			fmt.Fprintf(output, "Variables for profile %s: %s\n", input.Profile, strings.Join(input.Variables, ", "))
		}
	}
	for _, command := range plan.DeveloperCommands {
		if command.Default {
			fmt.Fprintf(output, "Developer command: %s (default)\n", command.Name)
			continue
		}
		fmt.Fprintf(output, "Developer command: %s\n", command.Name)
	}
	for _, name := range plan.ValidationCandidates {
		fmt.Fprintf(output, "Validation candidate: %s\n", name)
	}
	if len(plan.ValidationCommand) > 0 {
		command, _ := json.Marshal(plan.ValidationCommand)
		fmt.Fprintf(output, "Selected validation: %s\n", command)
	}
	fmt.Fprintf(output, "Package manager: %s\n", plan.PackageManager)
	for _, change := range plan.FileChanges {
		fmt.Fprintf(output, "File change: %s\n", change)
	}
	fmt.Fprintln(output, "Will write inject.toml:")
	fmt.Fprint(output, string(plan.ConfigData))
}

func detectPlaintextInputs(directory string) []PlaintextInput {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil
	}
	var names []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !isPlaintextInputName(name) {
			continue
		}
		names = append(names, name)
	}
	sort.Slice(names, func(left, right int) bool {
		if names[left] == ".env" {
			return true
		}
		if names[right] == ".env" {
			return false
		}
		return names[left] < names[right]
	})
	inputs := make([]PlaintextInput, 0, len(names))
	for _, name := range names {
		input := PlaintextInput{Name: name, Profile: "default", Selected: name == ".env"}
		if name != ".env" {
			input.Profile = strings.TrimPrefix(name, ".env.")
			if hasPlaintextInputName(names, ".env") {
				input.Precedence = ".env < " + name
			}
		}
		inputs = append(inputs, input)
	}
	return inputs
}

func selectPlaintextInputs(inputs []PlaintextInput, selected []string) error {
	if selected == nil {
		return nil
	}
	selectedNames := make(map[string]bool, len(selected)+1)
	for _, name := range selected {
		if selectedNames[name] {
			return fmt.Errorf("setup: plaintext input %q selected more than once", name)
		}
		selectedNames[name] = true
	}
	if hasPlaintextInput(inputs, ".env") && len(selectedNames) > 0 {
		selectedNames[".env"] = true
	}
	for index := range inputs {
		inputs[index].Selected = selectedNames[inputs[index].Name]
		delete(selectedNames, inputs[index].Name)
	}
	for name := range selectedNames {
		return fmt.Errorf("setup: plaintext input %q is not available", name)
	}
	return nil
}

func composeLocalProfiles(directory string, inputs []PlaintextInput) (map[string]map[string]string, error) {
	profiles := make(map[string]map[string]string)
	var base map[string]string
	for index := range inputs {
		if !inputs[index].Selected {
			continue
		}
		data, err := os.ReadFile(filepath.Join(directory, inputs[index].Name))
		if err != nil {
			return nil, fmt.Errorf("setup: read plaintext input %q: %w", inputs[index].Name, err)
		}
		secrets, err := parser.Parse(data)
		if err != nil {
			return nil, fmt.Errorf("setup: import plaintext input %q: %w", inputs[index].Name, err)
		}
		if inputs[index].Name == ".env" {
			base = secrets
		}
		complete := make(map[string]string, len(base)+len(secrets))
		for name, value := range base {
			complete[name] = value
		}
		for name, value := range secrets {
			complete[name] = value
		}
		inputs[index].Variables = sortedKeys(complete)
		profiles[inputs[index].Profile] = complete
	}
	return profiles, nil
}

func sortedKeys[Value any](values map[string]Value) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func isPlaintextInputName(name string) bool {
	if name == ".env" {
		return true
	}
	if !strings.HasPrefix(name, ".env.") || len(name) == len(".env.") {
		return false
	}
	excluded := map[string]bool{
		"example": true, "examples": true, "sample": true, "template": true,
		"tmpl": true, "dist": true, "backup": true, "bak": true, "old": true,
		"orig": true, "generated": true,
	}
	for _, part := range strings.Split(strings.TrimPrefix(name, ".env."), ".") {
		if excluded[strings.ToLower(part)] {
			return false
		}
	}
	return true
}

func hasPlaintextInput(inputs []PlaintextInput, name string) bool {
	for _, input := range inputs {
		if input.Name == name {
			return true
		}
	}
	return false
}

func hasPlaintextInputName(inputs []string, name string) bool {
	for _, input := range inputs {
		if input == name {
			return true
		}
	}
	return false
}

func isRuntimeScript(name string) bool {
	switch strings.ToLower(name) {
	case "dev", "start", "serve":
		return true
	default:
		return false
	}
}

func isFiniteValidationScript(name string) bool {
	switch strings.ToLower(name) {
	case "test", "check", "lint", "typecheck", "type-check", "build":
		return true
	default:
		return false
	}
}

func detectPackageManager(directory string) string {
	hasPackageManifest := false
	if data, err := os.ReadFile(filepath.Join(directory, "package.json")); err == nil {
		hasPackageManifest = true
		var manifest struct {
			PackageManager string `json:"packageManager"`
		}
		if json.Unmarshal(data, &manifest) == nil {
			family, _, _ := strings.Cut(manifest.PackageManager, "@")
			switch family {
			case "npm", "pnpm", "yarn", "bun":
				return family
			}
		}
	}
	for _, candidate := range []struct {
		file   string
		family string
	}{
		{file: "pnpm-lock.yaml", family: "pnpm"},
		{file: "yarn.lock", family: "yarn"},
		{file: "bun.lock", family: "bun"},
		{file: "bun.lockb", family: "bun"},
		{file: "package-lock.json", family: "npm"},
	} {
		if _, exists := statFile(filepath.Join(directory, candidate.file)); exists {
			return candidate.family
		}
	}
	if hasPackageManifest {
		return "npm"
	}
	return "not detected"
}

func hasRemoteReference(request Request) bool {
	return request.Account != "" || request.Vault != "" || request.ItemID != "" || request.Item != ""
}

func effectiveProvider(request Request) string {
	if request.Local {
		return "local"
	}
	if request.Provider != "" {
		return request.Provider
	}
	if hasRemoteReference(request) {
		return "1password"
	}
	return ""
}

func defaultProjectID(directory string) (string, error) {
	if data, err := os.ReadFile(filepath.Join(directory, "package.json")); err == nil {
		var manifest struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(data, &manifest) == nil && strings.TrimSpace(manifest.Name) != "" {
			return manifest.Name, nil
		}
	}
	if root, err := exec.Command("git", "-C", directory, "rev-parse", "--show-toplevel").Output(); err == nil {
		return filepath.Base(strings.TrimSpace(string(root))), nil
	}
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
	provider := effectiveProvider(request)
	if provider == "" {
		return fmt.Errorf("setup: source is required (local, 1password, or bitwarden)")
	}
	if provider != "local" && provider != "1password" && provider != "bitwarden" {
		return fmt.Errorf("setup: unsupported provider %q", request.Provider)
	}
	if request.Provider != "" && request.Local {
		return fmt.Errorf("setup: provider and local source cannot be selected together")
	}
	if request.Local {
		return nil
	}
	if provider == "1password" && (strings.TrimSpace(request.Account) == "" || strings.TrimSpace(request.Vault) == "") {
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

func plan(request Request, localProfiles map[string]map[string]string) (project.Config, error) {
	profile := project.Profile{
		Provider: effectiveProvider(request), Account: request.Account, Vault: request.Vault, ItemID: request.ItemID, Item: request.Item,
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
	if request.Local && len(localProfiles) > 0 {
		config.Profiles = make(map[string]project.Profile, len(localProfiles))
		for name := range localProfiles {
			config.Profiles[name] = project.Profile{Provider: "local"}
		}
	}
	if request.PackageScript == "" && request.Binding == "" {
		return config, nil
	}
	binding := request.Binding
	if binding == "" {
		binding = request.PackageScript
	}
	command := request.Command
	if request.PackageScript != "" {
		if err := validatePackageScript(request.Directory, request.PackageScript); err != nil {
			return project.Config{}, err
		}
		command = []string{"npm", "run", request.PackageScript}
	}
	if len(command) == 0 {
		return project.Config{}, fmt.Errorf("setup: binding %q requires a command", binding)
	}
	config.Commands[binding] = project.Binding{Command: command}
	return config, nil
}

func encodeConfig(config project.Config) ([]byte, error) {
	data := []byte(fmt.Sprintf("format_version = 1\nproject_id = %q\n", config.ProjectID))
	for _, name := range sortedKeys(config.Profiles) {
		profile := config.Profiles[name]
		data = append(data, fmt.Sprintf("\n[profiles.%s]\nprovider = %q\n", tomlKey(name), profile.Provider)...)
		if profile.Provider == "1password" {
			data = append(data, fmt.Sprintf("account = %q\nvault = %q\n", profile.Account, profile.Vault)...)
		}
		if profile.Provider == "local" {
			continue
		}
		if profile.ItemID != "" {
			data = append(data, fmt.Sprintf("item_id = %q\n", profile.ItemID)...)
		} else {
			data = append(data, fmt.Sprintf("item = %q\n", profile.Item)...)
		}
	}
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

func tomlKey(name string) string {
	if tomlBareKey.MatchString(name) {
		return name
	}
	return fmt.Sprintf("%q", name)
}

func validatePackageScript(directory, script string) error {
	data, err := os.ReadFile(filepath.Join(directory, "package.json"))
	if err != nil {
		return fmt.Errorf("setup: read package.json: %w", err)
	}
	var manifest struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("setup: invalid package.json: %w", err)
	}
	if _, exists := manifest.Scripts[script]; !exists {
		return fmt.Errorf("setup: package.json has no %q script", script)
	}
	return nil
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
	if _, exists := manifest.Scripts["dev"]; exists {
		candidates = append(candidates, "dev")
	}
	for name := range manifest.Scripts {
		if name != "dev" {
			candidates = append(candidates, name)
		}
	}
	if len(candidates) > 1 {
		sort.Strings(candidates[1:])
	}
	return candidates
}

func defaultPackageScript(candidates []string) string {
	if len(candidates) > 0 && candidates[0] == "dev" {
		return "dev"
	}
	return ""
}

func runValidation(request Request, localSecrets map[string]string) error {
	if request.RunValidation != nil {
		return request.RunValidation(request.Validate)
	}
	if request.Local {
		return executor.RunCommand(request.Validate, localSecrets)
	}
	if effectiveProvider(request) == "bitwarden" {
		provider, err := vaults.NewBitwardenProvider()
		if err != nil {
			return err
		}
		secrets, err := provider.Fetch(requestContext(), vaults.BitwardenReference{ItemID: request.ItemID, Item: request.Item})
		if err != nil {
			return err
		}
		return executor.RunCommand(request.Validate, secrets)
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
