package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func main() {
	if err := realMain(context.Background(), os.Args...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}

func realMain(ctx context.Context, args ...string) error {
	exec := args[0]

	targetCmd := &ffcli.Command{
		Name:      "target",
		ShortHelp: "Prints Zig target for GOOS/GOARCH or ZIGTARGET environment variables.",
		Exec: func(_ context.Context, _ []string) error {
			fmt.Println(target())
			return nil
		},
	}

	dumpCmdFunc := func(compiler string) *ffcli.Command {
		return &ffcli.Command{
			Name: compiler,
			Exec: func(_ context.Context, args []string) error {
				ztarget, err := target()
				if err != nil {
					return err
				}

				cmd := zigcmd("zig", compiler, ztarget, args...)
				fmt.Println(cmd)

				return nil
			},
		}
	}

	dumpCmd := &ffcli.Command{
		Name:      "dump",
		ShortHelp: "Prints Zig compile command to execute.",
		Subcommands: []*ffcli.Command{
			dumpCmdFunc("cc"),
			dumpCmdFunc("cxx"),
		},
		Exec: func(_ context.Context, args []string) error {
			ztarget, err := target()
			if err != nil {
				return err
			}

			cmd := zigcmd("zig", "cc", ztarget, args...)
			fmt.Println(cmd)
			return nil
		},
	}

	rootCmd := &ffcli.Command{
		ShortUsage:  fmt.Sprintf("%v <subcommand>", exec),
		Subcommands: []*ffcli.Command{targetCmd, dumpCmd},
		Exec: func(_ context.Context, args []string) error {
			return flag.ErrHelp
		},
	}

	return rootCmd.ParseAndRun(ctx, args[1:])
}

func target() (string, error) {
	goos := coalesce(os.Getenv("GOOS"), runtime.GOOS)
	goarch := coalesce(os.Getenv("GOARCH"), runtime.GOARCH)

	if zigtarget := os.Getenv("ZIGTARGET"); zigtarget != "" {
		return zigtarget, nil
	}

	return asZigTarget(goos, goarch)
}

func asZigTarget(goos, goarch string) (string, error) {
	switch goos {
	case "linux":
		switch goarch {
		case "amd64":
			return "x86_64-linux-musl", nil
		case "arm64":
			return "aarch64-linux-musl", nil
		case "arm":
			return "arm-linux-musleabi", nil
		default:
			return "", fmt.Errorf("unsupported arch: GOOS: %v GOARCH: %v", goos, goarch)
		}

	// TODO(selman): could not get it to work.
	// case "darwin":
	// 	switch goarch {
	// 	case "amd64":
	// 		return "x86_64-macos"
	// 	case "arm64":
	// 		return "aarch64-macos"
	// default:
	// 	panic(fmt.Sprintf("unsupported arch: GOOS: %v GOARCH: %v", goos, goarch))
	// }

	default:
		return "", fmt.Errorf("unsupported os: GOOS: %v GOARCH: %v", goos, goarch)
	}
}

func coalesce(v1, v2 string) string {
	if v1 == "" {
		return v2
	}

	return v1
}

func zigcmd(bin, cmd, target string, args ...string) string {
	zargs := []string{bin, cmd, "-target", target}
	zargs = append(zargs, args...)

	return strings.Join(
		zargs,
		" ",
	)
}
