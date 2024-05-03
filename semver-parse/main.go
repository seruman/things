package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

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
	scanner := bufio.NewScanner(stdin)
	for scanner.Scan() {
		line := scanner.Text()
		version, err := semver.NewVersion(line)
		if err != nil {
			fmt.Fprintf(stderr, "%v: %v\n", err, line)
			continue
		}

		dump(stdout, version)
	}

	return scanner.Err()
}

func dump(w io.Writer, v *semver.Version) {
	fmt.Fprintf(w, "Version: %s\n", v)
	fmt.Fprintf(w, "Major: %d\n", v.Major())
	fmt.Fprintf(w, "Minor: %d\n", v.Minor())
	fmt.Fprintf(w, "Patch: %d\n", v.Patch())
	fmt.Fprintf(w, "Prerelease: %s\n", v.Prerelease())
	fmt.Fprintf(w, "Meta: %s\n", v.Metadata())
}
