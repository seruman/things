package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func main() {
	if err := realMain(
		context.Background(),
		os.Stdin,
		os.Stdout,
		os.Args,
	); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain(
	ctx context.Context,
	stdin io.Reader,
	stdout io.Writer,
	args []string,
) error {
	exec := args[0]

	rootCmd := &ffcli.Command{
		ShortUsage: fmt.Sprintf("%v <subcommand>", exec),
		Exec: func(_ context.Context, _ []string) error {
			data := run(stdin)
			o, err := json.MarshalIndent(data, "", "  ")
			if err != nil {
				return err
			}

			fmt.Fprintf(stdout, "%s\n", o)
			return nil
		},
	}

	return rootCmd.ParseAndRun(ctx, args[1:])
}

func run(reader io.Reader) []map[string]string {
	var headers []string
	var data []map[string]string

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(headers) == 0 {
			headers = fields
			continue
		}

		row := make(map[string]string)
		for i, header := range headers {
			if i >= len(fields) {
				row[header] = ""
				continue
			}
			row[header] = fields[i]
		}
		data = append(data, row)
	}

	if len(data) == 0 {
		return []map[string]string{}
	}

	return data
}
