package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/harsh-sonkar/env-pull/cmd"
)

func main() {
	commandName := strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
	if commandName == "env-pull" {
		fmt.Fprintln(os.Stderr, "env-pull is deprecated; use inject instead")
	}
	cmd.Execute()
}
