package main

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/guptarohit/asciigraph"
	"github.com/mattn/go-isatty"
	promapi "github.com/prometheus/client_golang/api"
	promapiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prommodel "github.com/prometheus/common/model"

	"github.com/muesli/termenv"
	"golang.org/x/exp/maps"
	"golang.org/x/term"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := realMain(
		ctx,
		os.Args[0],
		os.Args[1:],
		os.Stdout,
	); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func realMain(
	ctx context.Context,
	exec string,
	args []string,
	stdout io.Writer,
) error {
	_ = exec
	_ = args

	// TODO(selman):
	query := args[0]
	addr := os.Getenv("PROMETHEUS_ADDR")

	client, err := promapi.NewClient(
		promapi.Config{
			Address: addr,
			RoundTripper: &loggingRoundTripper{
				roundTripper: http.DefaultTransport,
			},
		},
	)
	if err != nil {
		return err
	}
	api := promapiv1.NewAPI(client)

	now := time.Now()
	const duration = 7 * 24 * time.Hour

	// TODO(selman): copied from prometheus/web/ui
	resolution := math.Max(
		math.Floor(float64(duration.Milliseconds())/250000),
		1,
	)
	step := time.Duration(resolution) * time.Second

	range_ := promapiv1.Range{
		Start: now.Add(-duration),
		End:   now,
		Step:  step,
	}

	value, warnings, err := api.QueryRange(ctx, query, range_)
	if err != nil {
		return err
	}
	// TODO(selman): ignore for now
	_ = warnings

	result, ok := value.(prommodel.Matrix)
	if !ok {
		return fmt.Errorf("expected promapiv1.Matrix, got %T", value)
	}

	termout := termenv.NewOutput(stdout)
	f, ok := termout.TTY().(*os.File)
	if !ok || !isatty.IsTerminal(f.Fd()) {
		return fmt.Errorf("not a terminal")
	}

	termwidth, termheight, err := term.GetSize(int(f.Fd()))
	if err != nil {
		return err
	}
	// TODO(selman): Order of colors lost, re-runs are not consistent.
	allColors := maps.Values(asciigraph.ColorNames)

	// TODO(selman): This is 🤢. Find a better way.
	// Also, is there a way to eliminate bright colors those not seen well with
	// the background?
	var bgcolor asciigraph.AnsiColor
	var bgok bool
	switch c := termenv.BackgroundColor().(type) {
	case termenv.ANSIColor:
		bgcolor = asciigraph.AnsiColor(int(c))
		bgok = true
	case termenv.ANSI256Color:
		bgcolor = asciigraph.AnsiColor(int(c))
		bgok = true
	}

	if bgok {
		allColors = SliceFilter(allColors, func(color asciigraph.AnsiColor) bool {
			return color != bgcolor
		})
	}

	data := make([][]float64, len(result))
	colors := make([]asciigraph.AnsiColor, len(result))
	legends := make([]string, len(result))
	for i, sample := range result {
		samples := normalizeSerie(
			range_.Start,
			range_.End,
			*sample,
			step,
		)

		vs := SliceMap(samples, func(sample valueSample) float64 {
			return sample.Value
		})

		data[i] = vs
		color := allColors[i%len(allColors)]
		colors[i] = color
		legends[i] = fmt.Sprintf("%s>%s %v", color, asciigraph.Default, sample.Metric)
	}

	graph := asciigraph.PlotMany(
		data,
		asciigraph.Caption(query),
		// asciigraph.Precision(0),
		asciigraph.SeriesColors(colors...),
		asciigraph.Height(termheight/5),
		asciigraph.Width(termwidth-8),
	)

	fmt.Fprintln(termout, graph)
	// TODO(selman): Need to wrap depending on the width of the terminal.
	fmt.Fprintln(termout, strings.Join(legends, "\n"))

	return nil
}

type valueSample struct {
	Timestamp int64
	Value     float64
}

// TODO(selman): copied from prometheus/web/ui
func normalizeSerie(start time.Time, end time.Time, serie prommodel.SampleStream, resolution time.Duration) []valueSample {
	s := start.Unix()
	e := end.Unix()
	res := int64(resolution.Seconds())
	values := serie.Values

	var data []valueSample
	var valuePosition int

	for t := s; t < e; t += res {
		var currentValue *prommodel.SamplePair
		if len(values) > valuePosition {
			currentValue = &values[valuePosition]
		}

		if currentValue != nil && len(values) > valuePosition && int64(currentValue.Timestamp/1000) < t+res {
			v := float64(currentValue.Value)

			data = append(data, valueSample{
				Timestamp: int64(currentValue.Timestamp / 1000),
				Value:     v,
			})

			valuePosition++
		} else {
			data = append(data, valueSample{
				Timestamp: t,
				Value:     math.NaN(),
			})
		}
	}

	return data
}

func SliceMap[E, T any](s []E, f func(E) T) []T {
	r := make([]T, len(s))
	for i, e := range s {
		r[i] = f(e)
	}
	return r
}

func SliceFilter[E any](s []E, f func(E) bool) []E {
	r := make([]E, 0, len(s))
	for _, e := range s {
		if f(e) {
			r = append(r, e)
		}
	}
	return r
}

type loggingRoundTripper struct {
	roundTripper http.RoundTripper
}

func (r *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	d, _ := httputil.DumpRequest(req, true)
	fmt.Println(string(d))
	return r.roundTripper.RoundTrip(req)
}
