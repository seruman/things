package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
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
	flagset := flag.NewFlagSet("fakes3", flag.ExitOnError)
	flagset.SetOutput(stderr)

	var (
		flagAddr       string
		flagAutoBucket bool
	)
	flagset.StringVar(&flagAddr, "addr", "127.0.0.1:9000", "address to listen on")
	flagset.BoolVar(&flagAutoBucket, "auto-bucket", false, "create buckets on first use instead of returning NoSuchBucket")

	if err := flagset.Parse(osargs[1:]); err != nil {
		return err
	}

	faker := gofakes3.New(
		s3mem.New(),
		gofakes3.WithAutoBucket(flagAutoBucket),
		gofakes3.WithLogger(gofakes3.StdLog(log.New(stderr, "", log.LstdFlags), gofakes3.LogErr, gofakes3.LogWarn)),
	)

	ln, err := net.Listen("tcp", flagAddr)
	if err != nil {
		return err
	}

	srv := &http.Server{Handler: faker.Server()}

	errc := make(chan error, 1)
	go func() {
		errc <- srv.Serve(ln)
	}()

	fmt.Fprintf(stdout, "http://%s\n", ln.Addr())

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
