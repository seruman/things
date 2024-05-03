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
	flagQuote := fs.String("q", `"`, "quote character")
	fs.Parse(args[1:])

	scanner := bufio.NewScanner(in)

	for scanner.Scan() {
		line := scanner.Text()
		fmt.Printf("%s%s%s\n", *flagQuote, line, *flagQuote)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	fmt.Fprintln(out)

	return nil
}
