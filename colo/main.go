package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func main() {
	if err := realMain(
		os.Stdin,
		os.Stdout,
		os.Args,
	); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain(stdin io.Reader, stdout io.Writer, osargs []string) error {
	input := stdin
	if len(osargs) > 1 {
		input = strings.NewReader(strings.Join(osargs[1:], "\n"))
	}

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		hexColor := strings.TrimSpace(scanner.Text())
		if hexColor == "" {
			continue
		}
		if err := printCube(stdout, hexColor, 5); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("Error reading input: %v", err)
	}

	return nil
}

func printCube(w io.Writer, hexColor string, size int) error {
	hexColor = strings.TrimPrefix(hexColor, "#")
	if len(hexColor) != 6 {
		return fmt.Errorf("invalid hex color '%s'. Please provide a 6-digit hex color", hexColor)
	}

	r, err := strconv.ParseUint(hexColor[:2], 16, 8)
	if err != nil {
		return fmt.Errorf("invalid red color component in '%s': %v", hexColor, err)
	}
	g, err := strconv.ParseUint(hexColor[2:4], 16, 8)
	if err != nil {
		return fmt.Errorf("invalid green color component in '%s': %v", hexColor, err)
	}
	b, err := strconv.ParseUint(hexColor[4:], 16, 8)
	if err != nil {
		return fmt.Errorf("invalid blue color component in '%s': %v", hexColor, err)
	}

	colorCode := fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
	resetCode := "\x1b[0m"

	for range size {
		for range size {
			fmt.Fprint(w, colorCode+"  "+resetCode)
		}
		fmt.Fprintln(w)
	}

	return nil
}
