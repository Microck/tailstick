// Package platform abstracts OS-specific command execution and service management.
// It provides a Runner interface for shelling out to system commands and platform
// detection for Linux and Windows environments.
package platform

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type Runner struct {
	DryRun bool
}

func (r Runner) Run(ctx context.Context, args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("empty command")
	}
	if r.DryRun {
		return "[dry-run] " + strings.Join(args, " "), nil
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return out.String(), fmt.Errorf("%w: %s", err, strings.TrimSpace(out.String()))
	}
	return out.String(), nil
}
