package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	"github.com/olekukonko/tablewriter"
)

func main() {
	if err := realMain(
		context.Background(),
		os.Args,
		os.Stdin,
		os.Stdout,
	); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain(
	_ context.Context,
	args []string,
	stdin io.Reader,
	stdout io.Writer,
) error {
	fs := flag.NewFlagSet(args[0], flag.ExitOnError)
	flagFieldSeparator := fs.String("f", ",", "field separator")
	flagHeader := fs.Bool("h", true, "with header")

	err := fs.Parse(args[1:])
	if err != nil {
		return err
	}

	comma, _ := utf8.DecodeRuneInString(*flagFieldSeparator)
	if comma == utf8.RuneError {
		return fmt.Errorf("invalid field separator: utf8: %q", *flagFieldSeparator)
	}

	csvreader := csv.NewReader(stdin)
	csvreader.Comma = comma

	table, err := tablewriter.NewCSVReader(stdout, csvreader, *flagHeader)
	if err != nil {
		return err
	}

	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.Render()

	return nil
}
