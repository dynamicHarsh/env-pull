package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInvokingPackageManagerUsesRuntimeFamily(t *testing.T) {
	for _, manager := range []string{"npm", "pnpm", "yarn", "bun"} {
		t.Run(manager, func(t *testing.T) {
			directory := t.TempDir()
			path := filepath.Join(directory, manager)
			if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", directory)
			t.Setenv("npm_execpath", "")
			t.Setenv("npm_config_user_agent", manager+"/1.0.0")

			got, err := invokingPackageManager("npm")
			if err != nil {
				t.Fatalf("invokingPackageManager() error = %v", err)
			}
			if got != path {
				t.Errorf("invokingPackageManager() = %q, want %q", got, path)
			}
		})
	}
}

func TestInvokingPackageManagerFallsBackToProjectMetadata(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "pnpm")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory)
	t.Setenv("npm_execpath", "")
	t.Setenv("npm_config_user_agent", "")

	got, err := invokingPackageManager("pnpm")
	if err != nil {
		t.Fatalf("invokingPackageManager() error = %v", err)
	}
	if got != path {
		t.Errorf("invokingPackageManager() = %q, want %q", got, path)
	}
}

func TestInvokingPackageManagerRejectsMissingOrAmbiguousMetadata(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	tests := []struct {
		name      string
		execPath  string
		userAgent string
		fallback  string
		want      string
	}{
		{name: "missing", want: "package manager is not detected"},
		{name: "ambiguous", execPath: "/tools/pnpm", userAgent: "yarn/4.0.0", fallback: "npm", want: "package manager metadata is ambiguous"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("npm_execpath", test.execPath)
			t.Setenv("npm_config_user_agent", test.userAgent)
			_, err := invokingPackageManager(test.fallback)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("invokingPackageManager() error = %v, want error containing %q", err, test.want)
			}
		})
	}
}
