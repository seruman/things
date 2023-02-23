package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ijt/go-anytime"
)

const (
	formatUnixEpoch                 = "unix"
	formatRFC3339ishWithoutTimeZone = "2006-01-02 15:04:05"
)

func main() {
	if err := realMain(
		os.Args,
		os.Stdout,
	); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain(args []string, stdout io.Writer) error {
	exec := args[0]

	fs := flag.NewFlagSet(exec, flag.ExitOnError)
	flagFormat := fs.String("f", formatRFC3339ishWithoutTimeZone, "format")
	flagUTC := fs.Bool("u", true, "use UTC")
	flagNow := fs.String("n", "", "time to use as now")

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	fsargs := fs.Args()

	now := time.Now()
	if *flagNow != "" {
		n, err := anytime.Parse(*flagNow, now)
		if err != nil {
			return fmt.Errorf("now: %w", err)
		}
		now = n
	}

	if len(fsargs) == 0 {
		fs.Usage()
		return fmt.Errorf("missing argument")
	}
	t, err := anytime.Parse(fsargs[0], now)
	if err != nil {
		return err
	}

	if *flagUTC {
		t = t.UTC()
	}

	switch *flagFormat {
	case formatUnixEpoch:
		fmt.Fprintf(stdout, "%d\n", t.Unix())
	default:
		fmt.Fprintf(stdout, "%s\n", t.Format(*flagFormat))
	}

	return nil
}
