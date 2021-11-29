package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"
)

type DockerDesktop interface {
	Start(context.Context) error
	Stop(context.Context) error
}

var dockerDesktop DockerDesktop

func main() {
	if err := realMain(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain(ctx context.Context, args []string) error {
	exec := args[0]

	startCmd := &ffcli.Command{
		Name:       "start",
		ShortUsage: fmt.Sprintf("%v start", exec),
		ShortHelp:  "Starts Docker Desktop",
		Exec: func(ctx context.Context, args []string) error {
			return dockerDesktop.Start(ctx)
		},
	}

	stopCmd := &ffcli.Command{
		Name:       "stop",
		ShortUsage: fmt.Sprintf("%v stop", exec),
		ShortHelp:  "Stops Docker Desktop",
		Exec: func(ctx context.Context, args []string) error {
			return dockerDesktop.Stop(ctx)
		},
	}

	rootCmd := &ffcli.Command{
		ShortUsage:  fmt.Sprintf("%v <subcommand>", exec),
		Subcommands: []*ffcli.Command{startCmd, stopCmd},
		Exec: func(_ context.Context, args []string) error {
			return flag.ErrHelp
		},
	}

	return rootCmd.ParseAndRun(ctx, args[1:])
}
