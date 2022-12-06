package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/apognu/gocal"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/mergestat/timediff"
	"golang.org/x/exp/slog"
)

func main() {
	if err := realMain(os.Args, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)

	flagAll := flags.Bool("all", false, "")
	flagIstanbulTime := flags.Bool("istanbul-time", true, "")
	flagDuration := flags.Duration("within-next", 8*time.Hour, "")
	flagOut := flags.String("out", defaultOutputTmpl, "")

	err := flags.Parse(args[1:])
	if err != nil {
		return err
	}

	start := time.Now().UTC()
	end := start.Add(*flagDuration)

	if *flagAll {
		end = start.Add(365 * 24 * time.Hour)
	}

	matches, err := homeMatches(start, end)
	if err != nil {
		return err
	}

	if len(matches) == 0 {
		return nil
	}

	if *flagIstanbulTime {
		loc, _ := time.LoadLocation("Europe/Istanbul")
		for _, match := range matches {
			match.Start = match.Start.In(loc)
		}
	}

	tmpl, err := template.New("output").Parse(*flagOut)
	if err != nil {
		return fmt.Errorf("output: parse: %w", err)
	}

	data := struct {
		Matches  []*match
		Duration string
	}{
		Matches:  matches,
		Duration: timediff.TimeDiff(end, timediff.WithStartTime(start)),
	}
	err = tmpl.Execute(os.Stdout, data)
	if err != nil {
		return fmt.Errorf("output: execute: %w", err)
	}

	return nil
}

const (
	calendarURL       = "http://ics.fixtur.es/v2/home/fenerbahce.ics"
	defaultOutputTmpl = `🐤 *Traffic {{ .Duration }}!*
{{ range $val := .Matches -}}
*@{{ $val.When }}* with {{ $val.Away }}
{{ end }}`
)

type match struct {
	Start time.Time

	// Like i care 🤷
	Home, Away string
}

func (m *match) When() string {
	return m.Start.Format("2006-01-02 15:04")
}

func homeMatches(start, end time.Time) ([]*match, error) {
	calendar, err := fetchCalendar()
	if err != nil {
		return nil, fmt.Errorf("home-matches: %w", err)
	}

	cal := gocal.NewParser(calendar)
	cal.Start = &start
	cal.End = &end

	err = cal.Parse()
	if err != nil {
		return nil, fmt.Errorf("home-matches: calendar-parse: %w", err)
	}

	var matches []*match
	for _, event := range cal.Events {
		home, away := summary(event.Summary)
		matches = append(matches, &match{Home: home, Away: away, Start: *event.Start})
	}

	return matches, nil
}

func fetchCalendar() (*bytes.Reader, error) {
	httpclient := retryablehttp.NewClient()
	httpclient.RetryWaitMax = 5 * time.Minute
	httpclient.Logger = newSlogAdapter(slog.DebugLevel)

	resp, err := httpclient.Get(calendarURL)
	if err != nil {
		return nil, fmt.Errorf("fetch-calendar: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch-calendar: response: non-ok response, status: %v", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fetch-calendar: body: %w", err)
	}

	return bytes.NewReader(body), nil
}

func summary(s string) (string, string) {
	parts := strings.Split(s, " - ")

	return parts[0], parts[1]
}

type slogAdapter struct {
	*slog.Logger
}

var _ retryablehttp.LeveledLogger = (*slogAdapter)(nil)

func newSlogAdapter(level slog.Level) *slogAdapter {
	handleropts := slog.HandlerOptions{
		Level: level,
	}

	logger := slog.New(handleropts.NewTextHandler(os.Stderr))
	return &slogAdapter{logger}
}

func (r *slogAdapter) Error(msg string, keysAndValues ...interface{}) {
	r.Logger.Error(msg, nil, keysAndValues...)
}

func (r *slogAdapter) Info(msg string, keysAndValues ...interface{}) {
	r.Logger.Info(msg, keysAndValues...)
}

func (r *slogAdapter) Debug(msg string, keysAndValues ...interface{}) {
	r.Logger.Debug(msg, keysAndValues...)
}

func (r *slogAdapter) Warn(msg string, keysAndValues ...interface{}) {
	r.Logger.Warn(msg, keysAndValues...)
}
