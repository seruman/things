package main

import (
	"crypto/rand"
	"fmt"
	"os"

	"github.com/oklog/ulid/v2"
)

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain() error {
	u, err := ulid.New(ulid.Now(), rand.Reader)
	if err != nil {
		return err
	}

	fmt.Println(u.String())

	return nil
}
