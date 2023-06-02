package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func main() {
	if err := realMain(os.Args, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func realMain(args []string, stdout io.Writer) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: %s <cron expression>", args[0])
	}

	cron := args[1]

	description, err := DescribeCron(cron)
	if err != nil {
		return err
	}

	fmt.Fprintln(stdout, description)

	return nil
}

func DescribeCron(cron string) (string, error) {
	fields := strings.Fields(cron)

	if len(fields) != 5 {
		return "", fmt.Errorf("cron expression should have 5 fields")
	}

	minutes := fields[0]
	hours := fields[1]
	dayOfMonth := fields[2]
	month := fields[3]
	dayOfWeek := fields[4]

	switch {
	case minutes == "*" && hours == "*" && dayOfMonth == "*" && month == "*" && dayOfWeek == "*":
		return "This runs every minute.", nil
	case minutes == "*/15" && hours == "*" && dayOfMonth == "*" && month == "*" && dayOfWeek == "*":
		return "This runs every 15 minutes.", nil
	case minutes == "0" && hours == "*" && dayOfMonth == "*" && month == "*" && dayOfWeek == "*":
		return "This runs at the start of every hour.", nil
	case minutes == "0" && hours == "*/3" && dayOfMonth == "*" && month == "*" && dayOfWeek == "*":
		return "This runs every 3 hours.", nil
	case minutes == "0" && hours == "0" && dayOfMonth == "*" && month == "*" && dayOfWeek == "*":
		return "This runs every day at midnight.", nil
	case minutes == "0" && hours == "0" && dayOfMonth == "1" && month == "*" && dayOfWeek == "*":
		return "This runs every month at midnight on the first day.", nil
	case minutes == "0" && hours == "0" && dayOfMonth == "1" && month == "1" && dayOfWeek == "*":
		return "This runs every year at midnight on January 1.", nil
	}

	// Otherwise, construct the description
	description := "This runs "

	// Minutes
	if minutes == "*" {
		description += "every minute "
	} else {
		min, _ := strconv.Atoi(minutes)
		description += fmt.Sprintf("at %d minutes past the hour ", min)
	}

	// Hours
	if hours == "*" {
		description += "of every hour "
	} else {
		hr, _ := strconv.Atoi(hours)
		description += fmt.Sprintf("at %02d:00 ", hr)
	}

	// Day of Month
	if dayOfMonth == "*" {
		description += "on every day "
	} else {
		dom, _ := strconv.Atoi(dayOfMonth)
		description += fmt.Sprintf("on the %d of the month ", dom)
	}

	// Month
	if month == "*" {
		description += "of every month "
	} else {
		m, _ := strconv.Atoi(month)
		description += fmt.Sprintf("in month %d ", m)
	}

	// Day of Week
	if dayOfWeek == "*" {
		description += "and on every day of the week."
	} else {
		dow, _ := strconv.Atoi(dayOfWeek)
		description += fmt.Sprintf("and on the %d day of the week.", dow)
	}

	return description, nil
}
