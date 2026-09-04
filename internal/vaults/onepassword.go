package vaults

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/harsh-sonkar/env-pull/internal/parser"
)

type OnePasswordReference struct {
	Account string
	Vault   string
	ItemID  string
	Item    string
}

type commandRunner interface {
	Run(context.Context, ...string) ([]byte, error)
}

type onePasswordRunner struct {
	path string
}

func (runner onePasswordRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, runner.path, args...).Output()
}

type OnePasswordProvider struct {
	runner commandRunner
}

func NewOnePasswordProvider() (*OnePasswordProvider, error) {
	path, err := exec.LookPath("op")
	if err != nil {
		return nil, fmt.Errorf("1password: the op CLI is required; install it and run `op signin`")
	}
	return newOnePasswordProviderWithRunner(onePasswordRunner{path: path}), nil
}

func newOnePasswordProviderWithRunner(runner commandRunner) *OnePasswordProvider {
	return &OnePasswordProvider{runner: runner}
}

func (provider *OnePasswordProvider) Fetch(ctx context.Context, reference OnePasswordReference) (map[string]string, error) {
	item := reference.ItemID
	if item == "" {
		item = reference.Item
	}
	output, err := provider.runner.Run(ctx, "item", "get", item, "--vault", reference.Vault, "--account", reference.Account, "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("1password: unable to retrieve the configured secret note; run `op signin` and retry")
	}
	defer clear(output)

	var response struct {
		Fields []struct {
			ID    string `json:"id"`
			Value string `json:"value"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("1password: configured item is not a readable secure note")
	}
	for _, field := range response.Fields {
		if field.ID == "notesPlain" {
			return parser.Parse([]byte(field.Value))
		}
	}
	return nil, fmt.Errorf("1password: configured item does not contain a secure note")
}

func clear(data []byte) {
	for index := range data {
		data[index] = 0
	}
}
