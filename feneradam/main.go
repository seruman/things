package main

import (
	"log"

	"github.com/k0kubun/pp"
)

func main() {
	standings, err := MackolikFener()
	if err != nil {
		log.Fatal(err)
	}

	pp.Println(standings)
}
