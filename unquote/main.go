package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
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

func realMain(
	stdin io.Reader,
	stdout io.Writer,
	osargs []string,
) error {
	scanner := bufio.NewScanner(stdin)

	for scanner.Scan() {
		l := scanner.Text()
		v, err := strconv.Unquote(l)
		if err != nil {
			v = l
		}

		fmt.Fprintf(stdout, "%s\n", v)
	}

	return scanner.Err()
}
