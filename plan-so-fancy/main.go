package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
	"os"
	"strings"
)

func main() {
	//result, err := tfplanparse.Parse(os.Stdin)
	//if err != nil {
	//	panic(err)
	//}
	//
	//fmt.Println(result)
	//_, _ = pp.Println(result)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		color.Unset()
		line := scanner.Text()
		switch line {
		case "":
			fmt.Println(line)
		default:
			uncolored := uncolor(line)
			trimmed := strings.TrimSpace(uncolored)
			r := firstRune(trimmed)
			switch r {
			case '+':
				colored := color.GreenString(trimmed)
				a := strings.Replace(uncolored, trimmed, colored, -1)
				fmt.Println(a)
			case '-':
				colored := color.RedString(trimmed)
				a := strings.Replace(uncolored, trimmed, colored, -1)
				fmt.Println(a)
			case '~':
				colored := color.HiYellowString(trimmed)
				a := strings.Replace(uncolored, trimmed, colored, -1)
				fmt.Println(a)
			default:
				fmt.Println(line)
			}
		}

	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
}

func firstRune(str string) rune {
	for i, r := range str {
		if i == 0 {
			return r
		}
		return ' '
	}
	return ' '
}

func uncolor(in string) string {
	var out bytes.Buffer
	uncolorize := colorable.NewNonColorable(&out)
	_, _ = uncolorize.Write([]byte(in))

	return out.String()
}
