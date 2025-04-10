package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
)

func main() {
	if err := realMain(
		os.Args,
		os.Stdin,
		os.Stdout,
		os.Stderr,
	); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func realMain(
	_ []string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
) error {
	fs := flag.NewFlagSet("semver", flag.ExitOnError)
	flagCompact := fs.Bool("c", false, "compact output")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdin)
	for scanner.Scan() {
		line := scanner.Text()
		version, err := semver.NewVersion(line)
		if err != nil {
			fmt.Fprintf(stderr, "%v: %v\n", err, line)
			continue
		}

		dump(stdout, version, *flagCompact)
	}

	return scanner.Err()
}

func dump(w io.Writer, v *semver.Version, compact bool) {
	parts := []string{
		fmt.Sprintf("Version: %s", v),
		fmt.Sprintf("Major: %d", v.Major()),
		fmt.Sprintf("Minor: %d", v.Minor()),
		fmt.Sprintf("Patch: %d", v.Patch()),
		fmt.Sprintf("Prerelease: %s", v.Prerelease()),
		fmt.Sprintf("Meta: %s", v.Metadata()),
	}

	delimiter := "\n"
	if compact {
		delimiter = " "
	}

	fmt.Fprintln(w, strings.Join(parts, delimiter))
}
