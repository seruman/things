package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"

	"golang.org/x/mod/modfile"
)

func main() {
	if err := realMain(
		os.Stdin,
		os.Stdout,
		os.Args,
	); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain(stdin io.Reader, stdout io.Writer, osargs []string) error {
	// <tool> <file>
	fs := flag.NewFlagSet("gomoddump", flag.ExitOnError)

	if err := fs.Parse(osargs[1:]); err != nil {
		return err
	}

	args := fs.Args()

	if len(args) != 1 {
		fs.Usage()
		return fmt.Errorf("invalid number of arguments")
	}

	contents, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}

	f, err := modfile.Parse(args[0], contents, nil)
	if err != nil {
		return err
	}

	w := bufio.NewWriter(stdout)
	for _, r := range f.Require {
		fmt.Fprintf(w, "%s: %s@%s\n", args[0], r.Mod.Path, r.Mod.Version)
	}
	return w.Flush()
}
