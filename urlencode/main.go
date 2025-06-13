package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
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

func realMain(_ context.Context, stdin io.Reader, stdout io.Writer, _ io.Writer, _ []string) error {
	scanner := bufio.NewScanner(stdin)

	for scanner.Scan() {
		line := scanner.Text()
		encoded := url.QueryEscape(line)
		fmt.Fprintln(stdout, encoded)
	}

	return scanner.Err()
}
