package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/Masterminds/semver/v3"
	ansicolor "github.com/fatih/color"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func main() {
	if err := realMain(
		os.Args,
		os.Stdout,
		os.Stderr,
	); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func realMain(
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) error {
	exec := args[0]
	flagset := flag.NewFlagSet("git-semtag", flag.ExitOnError)
	flagPreRelease := flagset.Bool("pre-release", false, "return pre-release versions only")
	flagSortReverse := flagset.Bool("r", false, "sort in reverse order")
	flagForceColor := flagset.Bool("fc", false, "force color output")
	flagIgnoreInvalid := flagset.Bool("ii", false, "ignore invalid semver tags")

	flagset.Usage = func() {
		fmt.Fprintf(stderr, "usage: %s [options] [path]\n", exec)
		flagset.PrintDefaults()
	}

	err := flagset.Parse(args[1:])
	if err != nil {
		return err
	}
	_ = flagset.Args()

	if *flagForceColor {
		ansicolor.NoColor = false
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	r, err := git.PlainOpenWithOptions(cwd, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return err
	}

	tagrefs, err := r.Tags()
	if err != nil {
		return err
	}

	var tags []string
	err = tagrefs.ForEach(func(t *plumbing.Reference) error {
		tags = append(tags, t.Name().Short())
		return nil
	})
	if err != nil {
		return err
	}

	if len(tags) == 0 {
		return nil
	}

	var versions []semver.Version
	for _, r := range tags {
		v, err := semver.NewVersion(r)
		if err != nil {
			errstr := err.Error()
			if errors.Is(err, semver.ErrInvalidSemVer) {
				errstr = "invalid semver"
			}

			if !*flagIgnoreInvalid {
				fmt.Fprintf(stderr, "%v: %v\n", colorError.Sprintf("%v", errstr), r)
			}
			continue
		}

		versions = append(versions, *v)
	}

	sortFunc := func(a, b semver.Version) int {
		return b.Compare(&a)
	}

	if *flagSortReverse {
		sortFunc = func(a, b semver.Version) int {
			return a.Compare(&b)
		}
	}

	slices.SortFunc(versions, sortFunc)

	if *flagPreRelease {
		versions = filter(versions, func(v semver.Version) bool {
			return v.Prerelease() != ""
		})
	}

	for _, v := range versions {
		color := colorRelease
		if v.Prerelease() != "" {
			color = colorPreRelease
		}

		vs := color.Sprintf("%v", v.Original())

		fmt.Fprintln(stdout, vs)

	}

	return nil
}

func filter[T any](s []T, f func(T) bool) []T {
	var r []T
	for _, v := range s {
		if f(v) {
			r = append(r, v)
		}
	}
	return r
}

var (
	colorPreRelease = ansicolor.New(ansicolor.FgWhite, ansicolor.Italic)
	colorRelease    = ansicolor.New(ansicolor.Bold)
	colorError      = ansicolor.New(ansicolor.FgRed)
)
