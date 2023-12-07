package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
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
	fs := flag.NewFlagSet(args[0], flag.ExitOnError)
	fs.Parse(args[1:])

	scanner := bufio.NewScanner(in)

	var total float64
	var l int64
	for scanner.Scan() {
		l = +1
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		v, err := strconv.ParseFloat(line, 64)
		if err != nil {
			return fmt.Errorf("line# %v: %w", l, err)
		}

		total += v

	}

	if err := scanner.Err(); err != nil {
		return err
	}

	fmt.Fprintln(out, total)

	return nil
}
