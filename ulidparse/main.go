package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/oklog/ulid/v2"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := realMain(
		ctx,
		os.Stdin,
		os.Stdout,
		os.Stderr,
		os.Args,
	); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain(_ context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer, osargs []string) error {
	fs := flag.NewFlagSet("ulidparse", flag.ExitOnError)
	flagSort := fs.Bool("sort", false, "sort the output")
	fs.SetOutput(stderr)

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: %s [options] <query> [<file>]\n", fs.Name())
		fs.PrintDefaults()
	}

	if err := fs.Parse(osargs[1:]); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdin)

	var goterr bool
	for scanner.Scan() {
		line := scanner.Text()

		uu, err := ulid.Parse(line)
		if err != nil {
			goterr = true
			fmt.Fprintf(stderr, "error: %v\n", err)
		}

		if !*flagSort {
			fmt.Fprintf(stdout, "%s	%v\n", line, ulid.Time(uu.Time()).UTC())
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if goterr {
		return fmt.Errorf("some lines failed to parse")
	}

	return nil
}
