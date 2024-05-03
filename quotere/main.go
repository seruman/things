package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
)

func main() {
	if err := realMain(
		os.Args,
		os.Stdin,
		os.Stdout,
	); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func realMain(
	args []string,
	in io.Reader,
	out io.Writer,
) error {
	scanner := bufio.NewScanner(in)

	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintln(out, regexp.QuoteMeta(line))
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	fmt.Fprintln(out)

	return nil
}
