package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp/syntax"
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
	fs := flag.NewFlagSet("regesimp", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: %s [regexp ...]\n", fs.Name())
		fmt.Fprintln(fs.Output(), "With no arguments, read one regexp per line from stdin.")
		fs.PrintDefaults()
	}

	if err := fs.Parse(osargs[1:]); err != nil {
		return err
	}

	simplify := func(re string) error {
		parsed, err := syntax.Parse(re, syntax.Perl)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintln(stdout, parsed.Simplify().String())
		return err
	}

	if args := fs.Args(); len(args) > 0 {
		for _, re := range args {
			if err := simplify(re); err != nil {
				return err
			}
		}
		return nil
	}

	scanner := bufio.NewScanner(stdin)
	for scanner.Scan() {
		if err := simplify(scanner.Text()); err != nil {
			return err
		}
	}
	return scanner.Err()
}
