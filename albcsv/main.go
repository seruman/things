package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
)

type header int

const (
	type_ header = iota + 1
	time
	elb
	client_ip
	client_port
	target_ip
	target_port
	request_processing_time
	target_processing_time
	response_processing_time
	elb_status_code
	target_status_code
	received_bytes
	sent_bytes
	request_verb
	request_url
	request_proto
	user_agent
	ssl_cipher
	ssl_protocol
	target_group_arn
	trace_id
	domain_name
	chosen_cert_arn
	matched_rule_priority
	request_creation_time
	actions_executed
	redirect_url
	lambda_error_reason
	target_port_list
	target_status_code_list
	classification
	classification_reason
)

var headers = [...]header{
	type_,
	time,
	elb,
	client_ip,
	client_port,
	target_ip,
	target_port,
	request_processing_time,
	target_processing_time,
	response_processing_time,
	elb_status_code,
	target_status_code,
	received_bytes,
	sent_bytes,
	request_verb,
	request_url,
	request_proto,
	user_agent,
	ssl_cipher,
	ssl_protocol,
	target_group_arn,
	trace_id,
	domain_name,
	chosen_cert_arn,
	matched_rule_priority,
	request_creation_time,
	actions_executed,
	redirect_url,
	lambda_error_reason,
	target_port_list,
	target_status_code_list,
	classification,
	classification_reason,
}

var pattern = regexp.MustCompile(`^([^ ]*) ([^ ]*) ([^ ]*) ([^ ]*):([0-9]*) ([^ ]*)[:-]([0-9]*) ([-.0-9]*) ([-.0-9]*) ([-.0-9]*) (|[-0-9]*) (-|[-0-9]*) ([-0-9]*) ([-0-9]*) \"([^ ]*) (.*) (- |[^ ]*)\" \"([^\"]*)\" ([A-Z0-9-]+) ([A-Za-z0-9.-]*) ([^ ]*) \"([^\"]*)\" \"([^\"]*)\" \"([^\"]*)\" ([-.0-9]*) ([^ ]*) \"([^\"]*)\" \"([^\"]*)\" \"([^ ]*)\" \"([^s]+?)\" \"([^s]+)\" \"([^ ]*)\" \"([^ ]*)\"$`)

func main() {
	if err := realMain(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain(_ context.Context, _ []string) error {
	buf := bytes.NewBuffer(nil)
	csvwriter := csv.NewWriter(buf)

	headerrow := make([]string, len(headers))
	for i, h := range headers {
		headerrow[i] = h.String()
	}

	csvwriter.Write(headerrow)
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()

		matches := pattern.FindStringSubmatch(line)

		if len(matches) != len(headers)+1 {
			return fmt.Errorf("invalid line: %s", line)
		}

		row := make([]string, len(headers))
		for i, h := range headers {
			row[i] = matches[h]
			if row[i] == "-" {
				row[i] = ""
			}

		}

		csvwriter.Write(row)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	csvwriter.Flush()
	if err := csvwriter.Error(); err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, buf.String())

	return nil
}
