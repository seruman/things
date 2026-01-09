package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/lnquy/cron"
)

func main() {
	desc, err := cron.NewDescriptor(
		cron.Use24HourTimeFormat(true),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		result, err := desc.ToDescription(line, cron.Locale_en)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error parsing '%s': %v\n", line, err)
			continue
		}
		fmt.Println(result)
	}
}
