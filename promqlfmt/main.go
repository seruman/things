package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/prometheus/prometheus/promql/parser"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := realmain(
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

func realmain(
	_ context.Context,
	stdin io.Reader,
	stdout io.Writer,
	_ io.Writer,
	_ []string,
) error {
	b, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	expr, err := parser.ParseExpr(string(b))
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "%s\n", expr.Pretty(0))

	return err
}
