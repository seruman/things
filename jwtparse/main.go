package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

func main() {
	if err := realMain(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain(args []string) error {
	exec := args[0]
	fs := flag.NewFlagSet(exec, flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "output JSON")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: %s [flags] [jwt]\n", exec)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	tokens := fs.Args()
	if len(tokens) == 0 {
		s := bufio.NewScanner(os.Stdin)
		for s.Scan() {
			tokens = append(tokens, s.Text())
		}
		if err := s.Err(); err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
	}

	var parseErr error
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}

		jwt, err := parseJWT(token)
		if err != nil {
			parseErr = errors.Join(parseErr, fmt.Errorf("error parsing token: %w", err))
			continue
		}

		if *jsonOutput {
			v, _ := json.Marshal(jwt)
			fmt.Println(string(v))
		} else {
			fmt.Println(jwt.String())
		}
	}

	return parseErr
}

type JWT struct {
	Header    map[string]any `json:"header"`
	Payload   map[string]any `json:"payload"`
	Signature string         `json:"signature"`
}

func parseJWT(token string) (*JWT, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT: expected 3 parts, got %d", len(parts))
	}

	header, err := decodeSegment(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decoding header: %w", err)
	}

	payload, err := decodeSegment(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decoding payload: %w", err)
	}

	return &JWT{
		Header:    header,
		Payload:   payload,
		Signature: parts[2],
	}, nil
}

func decodeSegment(seg string) (map[string]any, error) {
	data, err := base64.RawURLEncoding.DecodeString(seg)
	if err != nil {
		data, err = base64.URLEncoding.DecodeString(seg)
		if err != nil {
			return nil, fmt.Errorf("base64 decode: %w", err)
		}
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}

	return result, nil
}

func (j *JWT) String() string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "=== Header ===\n")
	writeMap(&buf, j.Header)

	fmt.Fprintf(&buf, "\n=== Payload ===\n")
	writeMap(&buf, j.Payload)

	fmt.Fprintf(&buf, "\n=== Signature ===\n")
	fmt.Fprintf(&buf, "%s\n", j.Signature)

	return buf.String()
}

func writeMap(buf *bytes.Buffer, m map[string]any) {
	w := tabwriter.NewWriter(buf, 0, 2, 2, ' ', 0)
	for key, value := range m {
		formatted := formatValue(key, value)
		fmt.Fprintf(w, "  %s:\t%s\n", key, formatted)
	}
	w.Flush()
}

func formatValue(key string, value any) string {
	switch key {
	case "iat", "exp", "nbf", "auth_time":
		if num, ok := toFloat64(value); ok {
			t := time.Unix(int64(num), 0)
			return fmt.Sprintf("%v (%s)", value, t.Format(time.RFC3339))
		}
	}

	switch v := value.(type) {
	case []any:
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = fmt.Sprintf("%v", item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]any:
		data, _ := json.Marshal(v)
		return string(data)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int64:
		return float64(n), true
	case int:
		return float64(n), true
	default:
		return 0, false
	}
}
