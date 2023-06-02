package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
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
	flagDelim := fs.String("d", ",", "delimiter")
	fs.Parse(args[1:])

	scanner := bufio.NewScanner(in)
	first := true

	for scanner.Scan() {
		if first {
			first = false
		} else {
			fmt.Fprint(out, *flagDelim)
		}

		line := scanner.Text()
		fmt.Print(line)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	fmt.Fprintln(out)

	return nil
}
