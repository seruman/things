package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/google/uuid"
)

func main() {
	if err := realMain(os.Args, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain(args []string, w io.Writer) error {
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)
	flagNoDash := flags.Bool("no-dash", false, "")

	err := flags.Parse(args[1:])
	if err != nil {
		return err
	}

	v, err := uuid.NewRandom()
	if err != nil {
		return err
	}

	uuidstr := v.String()
	if *flagNoDash {
		uuidstr = strings.ReplaceAll(v.String(), "-", "")
	}

	fmt.Fprintln(w, uuidstr)

	return nil
}
