package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"

	"github.com/Masterminds/semver/v3"
	"github.com/olekukonko/tablewriter"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := realMain(
		ctx,
		os.Stdout,
		os.Stderr,
		os.Args,
	); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain(
	ctx context.Context,
	stdout io.Writer,
	stderr io.Writer,
	args []string,
) error {
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	dryRun := fs.Bool("dry", false, "print what would be upgraded without upgrading")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	pkgs, err := brewOutdated(ctx)
	if err != nil {
		return err
	}

	patchOnly := func(p *pkg) bool {
		return p.current.Major() == p.installed.Major() &&
			p.current.Minor() == p.installed.Minor()
	}

	notPinned := func(p *pkg) bool {
		return !p.pinned
	}

	formulae := filterz(mapz(pkgs.Formulae, toPkg), notNil)
	formulae = filterz(filterz(formulae, notPinned), patchOnly)

	casks := filterz(mapz(pkgs.Casks, toPkg), notNil)
	casks = filterz(filterz(casks, notPinned), patchOnly)

	if *dryRun {
		table := tablewriter.NewWriter(stdout)
		table.SetHeader([]string{"Type", "Name", "Installed", "Available"})
		table.SetBorder(false)
		table.SetColumnSeparator("")
		table.SetTablePadding("  ")
		table.SetNoWhiteSpace(true)
		table.SetHeaderLine(false)
		table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		for _, p := range formulae {
			table.Append([]string{"formula", p.name, p.installed.String(), p.current.String()})
		}
		for _, p := range casks {
			table.Append([]string{"cask", p.name, p.installed.String(), p.current.String()})
		}
		table.Render()
		return nil
	}

	if names := pkgNames(formulae); len(names) > 0 {
		if err := brewUpgrade(ctx, stdout, stderr, names...); err != nil {
			return fmt.Errorf("upgrading formulae: %w", err)
		}
	}

	if names := pkgNames(casks); len(names) > 0 {
		args := append([]string{"--cask"}, names...)
		if err := brewUpgrade(ctx, stdout, stderr, args...); err != nil {
			return fmt.Errorf("upgrading casks: %w", err)
		}
	}

	return nil
}

func pkgNames(ps []*pkg) []string {
	var result []string
	for _, p := range ps {
		result = append(result, p.name)
	}
	return result
}

func toPkg(p packageInfo) *pkg {
	if len(p.InstalledVersions) == 0 {
		return nil
	}

	installed := maybeSemver(p.InstalledVersions[0])
	if installed == nil {
		return nil
	}

	current := maybeSemver(p.CurrentVersion)
	if current == nil {
		return nil
	}

	return &pkg{
		name:      p.Name,
		installed: *installed,
		current:   *current,
		pinned:    p.Pinned,
	}
}

func notNil[T any](ptr *T) bool {
	return ptr != nil
}

type packageInfo struct {
	Name              string   `json:"name"`
	InstalledVersions []string `json:"installed_versions"`
	CurrentVersion    string   `json:"current_version"`
	Pinned            bool     `json:"pinned"`
}

type packages struct {
	Formulae []packageInfo `json:"formulae"`
	Casks    []packageInfo `json:"casks"`
}

type pkg struct {
	name      string
	installed semver.Version
	current   semver.Version
	pinned    bool
}

func brewOutdated(ctx context.Context) (packages, error) {
	cmd := exec.CommandContext(ctx, "brew", "outdated", "--json=v2")
	output, err := cmd.Output()
	if err != nil {
		var eerr *exec.ExitError
		if ok := errors.As(err, &eerr); ok {
			return packages{}, fmt.Errorf("brew command failed: %s", string(eerr.Stderr))
		}
		return packages{}, err
	}

	var pkgs packages
	if err := json.Unmarshal(output, &pkgs); err != nil {
		return packages{}, err
	}

	return pkgs, nil
}

func brewUpgrade(ctx context.Context, stdout io.Writer, stderr io.Writer, args ...string) error {
	cmdArgs := append([]string{"upgrade"}, args...)
	cmd := exec.CommandContext(ctx, "brew", cmdArgs...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func maybeSemver(v string) *semver.Version {
	sv, _ := semver.NewVersion(v)
	return sv
}

func filterz[T any](items []T, fn func(T) bool) []T {
	var result []T
	for _, item := range items {
		if fn(item) {
			result = append(result, item)
		}
	}
	return result
}

func mapz[T any, U any](items []T, fn func(T) U) []U {
	var result []U
	for _, item := range items {
		result = append(result, fn(item))
	}
	return result
}
