//go:build darwin

package main

import (
	"context"
	"fmt"
	"os/exec"
)

func init() {
	dockerDesktop = newDarwinDockerDesktop()
}

type darwinDockerDesktop struct{}

func newDarwinDockerDesktop() *darwinDockerDesktop {
	return &darwinDockerDesktop{}
}
func (d darwinDockerDesktop) Start(ctx context.Context) error {
	err := osascript(ctx, `tell application "Docker" to activate`)
	if err != nil {
		return fmt.Errorf("stop: %w", err)
	}

	return nil
}

func (d darwinDockerDesktop) Stop(ctx context.Context) error {
	err := osascript(ctx, `tell application "Docker" to quit`)
	if err != nil {
		return fmt.Errorf("stop: %w", err)
	}

	return nil
}

func osascript(ctx context.Context, command string) error {
	cmd := exec.CommandContext(ctx, "osascript", "-e", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript: %s: %w", output, err)
	}

	return nil
}
