package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func main() {
	if err := realMain(
		context.Background(),
		os.Args,
		os.Stdin,
		os.Stdout,
		os.Stderr,
	); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func realMain(
	ctx context.Context,
	args []string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
) error {
	exec := args[0]

	fs := flag.NewFlagSet(exec, flag.ExitOnError)
	flagCompact := fs.Bool("c", false, "compact output")

	bumpCmd := &ffcli.Command{
		Name:       "bump",
		ShortUsage: fmt.Sprintf("%v bump <patch|minor|major|prerelease>", exec),
		ShortHelp:  "Bump version by incrementing patch, minor, major, or prerelease",
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("bump requires exactly one argument: patch, minor, major, or prerelease")
			}

			bumpType := args[0]
			if bumpType != "patch" && bumpType != "minor" && bumpType != "major" && bumpType != "prerelease" {
				return fmt.Errorf("invalid bump type: %s (must be patch, minor, major, or prerelease)", bumpType)
			}

			scanner := bufio.NewScanner(stdin)
			for scanner.Scan() {
				line := scanner.Text()
				version, err := semver.NewVersion(line)
				if err != nil {
					fmt.Fprintf(stderr, "%v: %v\n", err, line)
					continue
				}

				var newVersion semver.Version
				switch bumpType {
				case "patch":
					newVersion = version.IncPatch()
				case "minor":
					newVersion = version.IncMinor()
				case "major":
					newVersion = version.IncMajor()
				case "prerelease":
					bumped, err := bumpPrerelease(version)
					if err != nil {
						fmt.Fprintf(stderr, "%v: %v\n", err, line)
						continue
					}
					newVersion = *bumped
				}

				fmt.Fprintln(stdout, newVersion.String())
			}

			return scanner.Err()
		},
	}

	rootCmd := &ffcli.Command{
		Name:        exec,
		ShortUsage:  fmt.Sprintf("%v [flags] [<subcommand>]", exec),
		FlagSet:     fs,
		Subcommands: []*ffcli.Command{bumpCmd},
		Exec: func(ctx context.Context, args []string) error {
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
		},
	}

	return rootCmd.ParseAndRun(ctx, args[1:])
}

func bumpPrerelease(v *semver.Version) (*semver.Version, error) {
	prerelease := v.Prerelease()

	if prerelease == "" && v.Metadata() == "" {
		// If no prerelease and no metadata, start a new prerelease cycle.
		// Increment patch and add -alpha.0.
		newVer := v.IncPatch()
		withPre, _ := newVer.SetPrerelease("alpha.0")
		return &withPre, nil
	}

	if prerelease == "" {
		return nil, fmt.Errorf("version has metadata but no prerelease to bump")
	}

	re := regexp.MustCompile(`^(alpha|beta|rc)(\.)?(\d+)$`)
	if matches := re.FindStringSubmatch(prerelease); matches != nil {
		prefix := matches[1]
		separator := matches[2] // Will be "." or empty
		num, _ := strconv.Atoi(matches[3])
		newPre := fmt.Sprintf("%s%s%d", prefix, separator, num+1)
		newVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d-%s", v.Major(), v.Minor(), v.Patch(), newPre))
		if v.Metadata() != "" {
			withMeta, _ := newVer.SetMetadata(v.Metadata())
			newVer = &withMeta
		}
		return newVer, nil
	}

	return nil, fmt.Errorf("prerelease '%s' does not match convention (alpha.N, beta.N, rc.N, alphaN, betaN, or rcN)", prerelease)
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
