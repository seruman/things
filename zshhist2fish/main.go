package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"os"
	"os/signal"
	"strings"

	"github.com/itchyny/zshhist-go"
	"gopkg.in/yaml.v3"
	"mvdan.cc/sh/v3/syntax"

	"github.com/seruman/babelfish/translate"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := realMain(
		ctx,
		os.Args,
		os.Stdin,
		os.Stdout,
		os.Stderr,
	); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func iterzhist(r *zshhist.Reader) iter.Seq2[int, zshhist.History] {
	return func(yield func(int, zshhist.History) bool) {
		i := 0
		for r.Scan() {
			i++
			if !yield(i, r.History()) {
				break
			}
		}
	}
}

func realMain(
	_ context.Context,
	_ []string,
	stdin *os.File,
	stdout *os.File,
	stderr *os.File,
) error {
	fs := flag.NewFlagSet("zshhist2fish", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: %s\n", fs.Name())
		fs.PrintDefaults()
	}

	flagDebug := fs.Bool("debug", false, "debug skipped stuff")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	logout := io.Discard
	if *flagDebug {
		logout = stderr
	}

	logger := slog.New(
		slog.NewTextHandler(
			logout,
			&slog.HandlerOptions{Level: slog.LevelDebug},
		),
	)

	r := zshhist.NewReader(stdin)
	p := syntax.NewParser(syntax.KeepComments(true), syntax.Variant(syntax.LangBash))

	var fishEntries []*entry
	for i, h := range iterzhist(r) {
		f, err := p.Parse(strings.NewReader(h.Command), "")
		if err != nil {
			logger.Debug("skip parse", "i", i, "command", h.Command, "err", err)
			continue
		}

		t := translate.NewTranslator()
		if err := t.File(f); err != nil {
			var uerr *translate.UnsupportedError
			if errors.As(err, &uerr) {
				logger.Debug("skip translate", "i", i, "command", h.Command, "err", err)
				continue
			}
		}

		var b bytes.Buffer
		_, _ = t.WriteTo(&b)
		fishEntries = append(fishEntries, &entry{
			When:    h.Time,
			Command: b.String(),
		})
	}

	if r.Err() != nil {
		return r.Err()
	}

	return yaml.NewEncoder(stdout).Encode(fishEntries)
}

type entry struct {
	When    int64  `yaml:"when"`
	Command string `yaml:"cmd"`
}
