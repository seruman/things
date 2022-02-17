package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/araddon/dateparse"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func main() {
	if err := realMain(
		context.Background(),
		os.Stdin,
		os.Stdout,
		os.Stderr,
		os.Args,
	); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain(
	ctx context.Context,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	args []string,
) error {
	exec := args[0]

	nowCmd := &ffcli.Command{
		Name:      "now",
		ShortHelp: "Current time in epoch seconds",
		Exec: func(_ context.Context, _ []string) error {
			fmt.Fprintln(stdout, time.Now().Unix())
			return nil
		},
	}

	decodeCmd := &ffcli.Command{
		Name:      "decode",
		ShortHelp: "Decode given epoch seconds input to human readable time",
		Exec: func(_ context.Context, _ []string) error {
			s := bufio.NewScanner(stdin)
			for s.Scan() {
				t, err := dateparse.ParseAny(s.Text())
				if err != nil {
					fmt.Fprintf(stderr, "ERROR: %v\n", err)
					continue
				}

				fmt.Fprintln(stdout, t.UTC().Format(time.RFC3339))
			}

			return nil
		},
	}

	encodeCmd := &ffcli.Command{
		Name:      "encode",
		ShortHelp: "Encode given human readable time to epoch seconds",
		Exec: func(_ context.Context, _ []string) error {
			s := bufio.NewScanner(stdin)
			for s.Scan() {
				t, err := dateparse.ParseAny(s.Text())
				if err != nil {
					fmt.Fprintf(stderr, "ERROR: %v\n", err)
					continue
				}

				fmt.Fprintln(stdout, t.UTC().Unix())
			}

			return nil
		},
	}

	rootCmd := &ffcli.Command{
		ShortUsage:  fmt.Sprintf("%v <subcommand>", exec),
		Subcommands: []*ffcli.Command{nowCmd, decodeCmd, encodeCmd},
		Exec: func(_ context.Context, args []string) error {
			return flag.ErrHelp
		},
	}

	return rootCmd.ParseAndRun(ctx, args[1:])
}
