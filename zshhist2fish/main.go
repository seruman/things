package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
)

// TODO: add best effort transpilation;
// - && || -> and or
// - $(...) -> (...)
// - <(...) -> (... | psub)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := realMain(
		ctx,
		os.Stdin,
		os.Stdout,
		os.Stderr,
		os.Args,
	); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain(
	_ context.Context,
	stdin *os.File,
	stdout *os.File,
	_ *os.File,
	_ []string,
) error {
	entries, err := ParseHistory(stdin)
	if err != nil {
		return fmt.Errorf("parse-history: %w", err)
	}

	if err := WriteHistory(stdout, entries); err != nil {
		return fmt.Errorf("write-history: %w", err)
	}

	return nil
}

type entry struct {
	timestamp int64
	command   string
}

type decoder struct {
	r *bufio.Reader
}

func newDecoder(r io.Reader) *decoder {
	return &decoder{r: bufio.NewReader(r)}
}

func (d *decoder) zshEntry() ([]byte, error) {
	line, err := d.r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	result := make([]byte, 0, len(line))

	// Ripped from;
	// https://gist.github.com/xkikeg/4162343/3ed7dfc1147b56931c4910cab0da47076b435d0d
	change := false
	for _, b := range line {
		if b == 0x83 {
			change = true
			continue
		}

		if change {
			result = append(result, b^32)
		} else {
			result = append(result, b)
		}
		change = false
	}
	return result, nil
}

func parseEntry(line string) (*entry, error) {
	// : 1736681683:0;fd
	// entry := &entry{ timestamp: 1736681683, command: "fd" }
	if !strings.HasPrefix(line, ": ") {
		return nil, nil
	}

	line = strings.TrimPrefix(line, ": ")
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("missing timestamp separator: %s", line)
	}

	timestamp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing timestamp: %w, %s", err, line)
	}

	parts = strings.SplitN(parts[1], ";", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("missing command separator: %s", line)
	}

	return &entry{
		timestamp: timestamp,
		command:   strings.TrimSuffix(parts[1], "\n"),
	}, nil
}

func ParseHistory(r io.Reader) ([]*entry, error) {
	d := newDecoder(r)
	var entries []*entry

	for {
		line, err := d.zshEntry()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read-zsh-entry: %w", err)
		}

		lineStr := string(line)
		if strings.HasPrefix(lineStr, ": ") {
			entry, err := parseEntry(lineStr)
			if err != nil {
				continue
			}
			if entry != nil {
				entries = append(entries, entry)
			}
			continue
		}

		if len(entries) > 0 {
			entries[len(entries)-1].command += lineStr
		}
	}

	for _, e := range entries {
		e.command = strings.TrimSuffix(e.command, "\n")
	}

	return entries, nil
}

func formatFishEntry(e *entry) string {
	return fmt.Sprintf("- cmd: %s\n  when: %d", e.command, e.timestamp)
}

func WriteHistory(w io.Writer, entries []*entry) error {
	for i, e := range entries {
		if i > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(w, formatFishEntry(e)); err != nil {
			return err
		}
	}
	return nil
}
