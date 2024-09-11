package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
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

func realMain(stdin io.Reader, stdout io.Writer, osargs []string) error {
	fs := flag.NewFlagSet("rivers", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: %s <regexp> <template>\n", fs.Name())
		fs.PrintDefaults()
	}

	if err := fs.Parse(osargs[1:]); err != nil {
		return err
	}

	args := fs.Args()

	if len(args) != 2 {
		fs.Usage()
		return fmt.Errorf("invalid number of arguments")
	}

	re := args[0]
	template := args[1]

	pattern, err := regexp.Compile(re)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdin)
	for scanner.Scan() {
		line := scanner.Text()

		var result []byte
		for _, submatch := range pattern.FindAllStringSubmatchIndex(line, -1) {
			result = pattern.ExpandString(result, template, line, submatch)
		}

		fmt.Fprintf(stdout, "%s\n", result)
	}

	return scanner.Err()
}
