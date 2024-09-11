package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

func main() {
	if err := realMain(
		context.Background(),
		os.Stdin,
		os.Stdout,
		os.Args,
	); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func realMain(
	ctx context.Context,
	stdin io.Reader,
	stdout io.Writer,
	args []string,
) error {
	fs := flag.NewFlagSet("promlabels", flag.ExitOnError)
	flagJSON := fs.Bool("json", false, "output in JSON format")

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdin)

	for scanner.Scan() {
		line := scanner.Text()
		metricselector, err := parser.ParseMetricSelector(line)
		if err != nil {
			return fmt.Errorf("parse-metric-selector: %w", err)
		}

		for _, matcher := range metricselector {
			if *flagJSON {
				m := (*Matcher)(matcher)
				b, err := json.Marshal(m)
				if err != nil {
					return err
				}

				fmt.Fprintf(stdout, "%s\n", b)
				continue
			}

			fmt.Fprintf(stdout, "%s\n", matcher)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

type Matcher labels.Matcher

func (m Matcher) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(struct {
		Name  string `json:"name"`
		Value string `json:"value"`
		Type  string `json:"type"`
	}{
		Type:  m.Type.String(),
		Name:  m.Name,
		Value: m.Value,
	})

	return b, err
}
