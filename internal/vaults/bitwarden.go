package vaults

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/harsh-sonkar/env-pull/internal/parser"
)

type BitwardenReference struct {
	ItemID string
	Item   string
}

type BitwardenProvider struct {
	runner commandRunner
}

func NewBitwardenProvider() (*BitwardenProvider, error) {
	path, err := exec.LookPath("bw")
	if err != nil {
		return nil, fmt.Errorf("bitwarden: the bw CLI is required; install it and run `bw login`")
	}
	return newBitwardenProviderWithRunner(cliRunner{path: path}), nil
}

func newBitwardenProviderWithRunner(runner commandRunner) *BitwardenProvider {
	return &BitwardenProvider{runner: runner}
}

func (provider *BitwardenProvider) Fetch(ctx context.Context, reference BitwardenReference) (map[string]string, error) {
	item := reference.ItemID
	if item == "" {
		item = reference.Item
	}
	output, err := provider.runner.Run(ctx, "get", "notes", item)
	if err != nil {
		return nil, fmt.Errorf("bitwarden: unable to retrieve the configured secret note; run `bw login` and retry")
	}
	defer clear(output)

	return parser.Parse(output)
}
