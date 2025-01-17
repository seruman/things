package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dustin/go-humanize"
)

func main() {
	if err := realMain(
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
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	osargs []string,
) error {
	input := stdin
	if len(osargs) > 1 {
		input = strings.NewReader(strings.Join(osargs[1:], "\n"))
	}

	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
		line := scanner.Text()
		b, err := humanize.ParseBytes(line)
		if err != nil {
			fmt.Fprintf(stderr, "parse: %v:  %v\n", line, err)
			continue
		}

		fmt.Fprintln(stdout, humanize.Bytes(b))

	}

	return scanner.Err()
}
