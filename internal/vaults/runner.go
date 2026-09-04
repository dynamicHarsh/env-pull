package vaults

import (
	"context"
	"os/exec"
)

type cliRunner struct {
	path string
}

func (runner cliRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, runner.path, args...).Output()
}
