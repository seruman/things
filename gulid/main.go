package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/oklog/ulid/v2"
)

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain() error {
	entropy := rand.New(rand.NewSource(time.Now().UnixNano()))
	ms := ulid.Timestamp(time.Now())

	u, err := ulid.New(ms, entropy)
	if err != nil {
		return err
	}

	fmt.Println(u.String())

	return nil
}
