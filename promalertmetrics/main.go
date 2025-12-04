package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/prometheus/prometheus/promql/parser"
	"github.com/prometheus/prometheus/rules"
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

func realMain(
	_ context.Context,
	_ *os.File,
	stdout *os.File,
	stderr *os.File,
	args []string,
) error {
	fs := flag.NewFlagSet(args[0], flag.ExitOnError)
	showFileNames := fs.Bool("f", false, "Show file names before each series")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: %s [options] <rules-files>\n", args[0])
		fs.PrintDefaults()
	}

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		fs.Usage()
		return fmt.Errorf("expected at least one rules file")
	}

	filepaths := fs.Args()

	type result struct {
		file   string
		series string
	}

	var results []result

	loader := rules.FileLoader{}
	for _, path := range filepaths {
		groups, errs := loader.Load(path, true)
		if len(errs) > 0 {
			return errors.Join(errs...)
		}

		for _, group := range groups.Groups {
			for _, rule := range group.Rules {
				expr, err := loader.Parse(rule.Expr)
				if err != nil {
					return fmt.Errorf("failed to parse expression %q: %w", rule.Expr, err)
				}

				parser.Inspect(expr, func(node parser.Node, _ []parser.Node) error {
					vs, ok := node.(*parser.VectorSelector)
					if !ok {
						return nil
					}

					if vs.Name == "" {
						log.Fatalf("missing name: %v", rule.Alert)
					}

					results = append(results, result{
						file:   path,
						series: vs.Name,
					})

					return nil
				})
			}
		}

	}

	for r := range results {
		if *showFileNames {
			fmt.Fprintf(stdout, "%v:\n  %v\n", results[r].file, results[r].series)
		}

		fmt.Fprintf(stdout, "%v\n", results[r].series)
	}

	return nil
}
