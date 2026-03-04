package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"

	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kversion"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := realMain(ctx, os.Args, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain(ctx context.Context, osargs []string, stdout io.Writer, stderr io.Writer) error {
	flagset := flag.NewFlagSet("kfake", flag.ExitOnError)
	flagset.SetOutput(stderr)

	var (
		flagLogLevel   string
		flagVersion    string
		flagPprofAddr  string
		flagPorts      string
		flagSeedTopics string
		flagBcfgs      = make(brokerConfigFlag)
	)

	flagset.StringVar(&flagLogLevel, "log-level", "none", "log level: none|error|warn|info|debug")
	flagset.StringVar(&flagLogLevel, "l", "none", "log level (shorthand)")
	flagset.StringVar(&flagVersion, "as-version", "", "Kafka version to emulate (e.g. 2.8, 3.5)")
	flagset.StringVar(&flagPprofAddr, "pprof", "", "pprof port on 127.0.0.1 (e.g. :6060), empty to disable")
	flagset.StringVar(&flagPorts, "ports", "9092,9093,9094", "broker ports (comma-separated)")
	flagset.StringVar(&flagSeedTopics, "seed-topics", "foo", "topics to seed (comma-separated)")
	flagset.Var(flagBcfgs, "broker-config", "broker config key=value (repeatable)")
	flagset.Var(flagBcfgs, "c", "broker config key=value (shorthand, repeatable)")

	if err := flagset.Parse(osargs[1:]); err != nil {
		return err
	}

	ports, err := parsePorts(flagPorts)
	if err != nil {
		return err
	}

	seedTopics := parseCSV(flagSeedTopics)
	if len(seedTopics) == 0 {
		seedTopics = []string{"foo"}
	}

	logLevel, err := parseLogLevel(flagLogLevel)
	if err != nil {
		return err
	}

	if flagPprofAddr != "" {
		addr := net.JoinHostPort("127.0.0.1", strings.TrimPrefix(flagPprofAddr, ":"))
		go func() {
			fmt.Fprintf(stderr, "pprof listening on %s\n", addr)
			if err := http.ListenAndServe(addr, nil); err != nil {
				fmt.Fprintf(stderr, "pprof failed: %v\n", err)
			}
		}()
	}

	opts := []kfake.Opt{
		kfake.Ports(ports...),
		kfake.SeedTopics(-1, seedTopics...),
		kfake.WithLogger(kfake.BasicLogger(stderr, logLevel)),
	}

	if flagVersion != "" {
		v := kversion.FromString(flagVersion)
		if v == nil {
			return fmt.Errorf("unknown version %q; valid versions: %v", flagVersion, kversion.VersionStrings())
		}
		opts = append(opts, kfake.MaxVersions(v))
	}

	if len(flagBcfgs) > 0 {
		opts = append(opts, kfake.BrokerConfigs(flagBcfgs))
	}

	cluster, err := kfake.NewCluster(opts...)
	if err != nil {
		return err
	}
	defer cluster.Close()

	fmt.Fprintln(stdout, strings.Join(cluster.ListenAddrs(), ","))

	<-ctx.Done()
	return nil
}

type brokerConfigFlag map[string]string

func (f brokerConfigFlag) String() string {
	if len(f) == 0 {
		return ""
	}

	keys := make([]string, 0, len(f))
	for k := range f {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, k+"="+f[k])
	}

	return strings.Join(pairs, ",")
}

func (f brokerConfigFlag) Set(s string) error {
	k, v, ok := strings.Cut(s, "=")
	if !ok {
		return fmt.Errorf("expected key=value, got %q", s)
	}
	f[k] = v
	return nil
}

func parseLogLevel(logLevel string) (kfake.LogLevel, error) {
	switch strings.ToLower(logLevel) {
	case "debug":
		return kfake.LogLevelDebug, nil
	case "info":
		return kfake.LogLevelInfo, nil
	case "warn":
		return kfake.LogLevelWarn, nil
	case "error":
		return kfake.LogLevelError, nil
	case "none":
		return kfake.LogLevelNone, nil
	default:
		return kfake.LogLevelNone, fmt.Errorf("invalid log level %q (expected: none|error|warn|info|debug)", logLevel)
	}
}

func parsePorts(input string) ([]int, error) {
	parts := parseCSV(input)
	if len(parts) == 0 {
		return nil, fmt.Errorf("at least one broker port is required")
	}

	ports := make([]int, 0, len(parts))
	for _, p := range parts {
		v, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid port %q: %w", p, err)
		}
		if v <= 0 || v > 65535 {
			return nil, fmt.Errorf("invalid port %d: must be between 1 and 65535", v)
		}
		ports = append(ports, v)
	}

	return ports, nil
}

func parseCSV(input string) []string {
	parts := strings.Split(input, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}
