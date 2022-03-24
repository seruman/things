package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	urlpkg "net/url"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func main() {
	if err := realMain(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain(ctx context.Context, args []string) error {
	exec := args[0]
	fs := flag.NewFlagSet(exec, flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "output JSON")

	rootCmd := &ffcli.Command{
		FlagSet:    fs,
		ShortUsage: fmt.Sprintf("%v <subcommand>", exec),
		Exec: func(_ context.Context, _ []string) error {
			s := bufio.NewScanner(os.Stdin)

			for s.Scan() {
				line := s.Text()
				u, err := urlpkg.Parse(line)
				if err != nil {
					log.Fatal(err)
				}

				var keys []string
				for key := range u.Query() {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				query := make(map[string]string)
				for _, key := range keys {
					query[key] = u.Query().Get(key)
				}

				uu := url{
					Scheme:   u.Scheme,
					Host:     u.Hostname(),
					Port:     u.Port(),
					User:     u.User.String(),
					Path:     u.Path,
					Query:    query,
					Fragment: u.Fragment,
				}

				if *jsonOutput {
					v, _ := json.Marshal(uu)
					fmt.Println(string(v))
				} else {
					fmt.Println(uu.String())
				}
			}

			return nil
		},
	}

	return rootCmd.ParseAndRun(ctx, args[1:])
}

type url struct {
	Scheme   string            `json:"scheme"`
	Host     string            `json:"host"`
	Port     string            `json:"port"`
	User     string            `json:"user"`
	Path     string            `json:"path"`
	Query    map[string]string `json:"query"`
	Fragment string            `json:"fragment"`
}

func (u *url) String() string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "Scheme: %v\n", u.Scheme)
	fmt.Fprintf(&buf, "Host: %v\n", u.Host)
	fmt.Fprintf(&buf, "Port: %v\n", u.Port)
	fmt.Fprintf(&buf, "User: %v\n", u.User)
	fmt.Fprintf(&buf, "Path: %v\n", u.Path)

	fmt.Fprintf(&buf, "Query:\n")

	w := tabwriter.NewWriter(&buf, 0, 2, 2, ' ', 0)
	for key, value := range u.Query {
		fmt.Fprintf(w, "  %v:\t%v\n", key, value)
	}
	w.Flush()
	fmt.Fprintf(&buf, "Fragment: %v\n", u.Fragment)

	return buf.String()
}
