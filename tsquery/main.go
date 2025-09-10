package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"slices"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/dockerfile"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/hcl"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

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

func realMain(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer, osargs []string) error {
	fs := flag.NewFlagSet("tsquery", flag.ExitOnError)
	fs.SetOutput(stderr)

	var (
		flagLang             string
		flagCaptues          stringsFlag
		flagWithCaptureNames bool
	)

	fs.StringVar(&flagLang, "l", "go", "language")
	fs.Var(&flagCaptues, "c", "captures to output (comma-separated)")
	fs.BoolVar(&flagWithCaptureNames, "n", false, "output capture names")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: %s [options] <query> [<file>]\n", fs.Name())
		fs.PrintDefaults()
	}

	if err := fs.Parse(osargs[1:]); err != nil {
		return err
	}

	args := fs.Args()

	in := stdin
	var query string

	switch len(args) {
	case 1:
		query = args[0]
	case 2:
		f, err := os.Open(args[1])
		if err != nil {
			return err
		}

		defer f.Close()
		in = f
		query = args[0]
	default:
		fs.Usage()
		return fmt.Errorf("invalid number of arguments")
	}

	lang := golang.GetLanguage()
	switch flagLang {
	case "go":
		lang = golang.GetLanguage()
	case "java":
		lang = java.GetLanguage()
	case "python":
		lang = python.GetLanguage()
	case "javascript":
		lang = javascript.GetLanguage()
	case "typescript":
		lang = typescript.GetLanguage()
	case "tsx":
		lang = tsx.GetLanguage()
	case "hcl":
		lang = hcl.GetLanguage()
	case "dockerfile":
		lang = dockerfile.GetLanguage()
	default:
		return fmt.Errorf("unsupported language: %s", flagLang)
	}

	dump := func(capture string, value string) {
		fmt.Fprintln(stdout, value)
	}

	if flagWithCaptureNames {
		dump = func(capture string, value string) {
			fmt.Fprintf(stdout, "%s: %s\n", capture, value)
		}
	}

	src, err := io.ReadAll(in)
	if err != nil {
		return err
	}

	node, err := sitter.ParseCtx(ctx, src, lang)
	if err != nil {
		return err
	}

	q, err := sitter.NewQuery([]byte(query), lang)
	if err != nil {
		return err
	}

	cursor := sitter.NewQueryCursor()
	cursor.Exec(q, node)
	for {
		m, ok := cursor.NextMatch()
		if !ok {
			break
		}

		m = cursor.FilterPredicates(m, src)
		for _, c := range m.Captures {
			capture := q.CaptureNameForId(c.Index)
			if len(flagCaptues) > 0 && !slices.Contains(flagCaptues, capture) {
				continue
			}

			value := c.Node.Content(src)
			dump(capture, value)
		}
	}

	return nil
}

type stringsFlag []string

func (sf *stringsFlag) String() string {
	return strings.Join(*sf, ",")
}

func (sf *stringsFlag) Set(value string) error {
	for _, s := range strings.Split(value, ",") {
		*sf = append(*sf, strings.TrimSpace(s))
	}
	return nil
}
