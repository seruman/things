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
	buf := make([]byte, 0, 64*1024)
	scanner := bufio.NewScanner(stdin)
	scanner.Buffer(buf, 10*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		encoded, err := url.QueryUnescape(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error decoding URL: %s\n", err)
		}
		fmt.Fprintln(stdout, encoded)
	}

	return scanner.Err()
}
