package main

import (
	"context"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	if err := realMain(
		context.Background(),
		os.Args,
		os.Stdout,
	); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain(
	_ context.Context,
	osargs []string,
	stdout io.Writer,
) error {
	flagset := flag.NewFlagSet("csv", flag.ExitOnError)
	flagNoHeader := flagset.Bool("nh", false, "do not output header")
	flagNoDash := flagset.Bool("nd", false, "replace dashes with empty strings")

	if err := flagset.Parse(osargs[1:]); err != nil {
		return err
	}

	csvreader := csv.NewReader(os.Stdin)
	csvreader.Comma = ' '

	csvwriter := csv.NewWriter(stdout)
	if !*flagNoHeader {
		if err := csvwriter.Write(header); err != nil {
			return err
		}
	}

	for {
		record, err := csvreader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return err
		}

		if *flagNoDash {
			for i, field := range record {
				if field == "-" {
					record[i] = ""
				}
			}
		}

		if err := csvwriter.Write(record); err != nil {
			return err
		}
	}

	csvwriter.Flush()
	return csvwriter.Error()
}

var header = []string{
	"type",
	"time",
	"elb",
	"client_port",
	"target_port",
	"request_processing_time",
	"target_processing_time",
	"response_processing_time",
	"elb_status_code",
	"target_status_code",
	"received_bytes",
	"sent_bytes",
	"request",
	"user_agent",
	"ssl_cipher",
	"ssl_protocol",
	"target_group_arn",
	"trace_id",
	"domain_name",
	"chosen_cert_arn",
	"matched_rule_priority",
	"request_creation_time",
	"actions_executed",
	"redirect_url",
	"error_reason",
	"target:port_list",
	"target_status_code_list",
	"classification",
	"classification_reason",
}
