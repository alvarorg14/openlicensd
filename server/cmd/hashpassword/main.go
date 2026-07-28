package main

import (
	"fmt"
	"os"

	"github.com/openlicensd/openlicensd/server/internal/auth"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: hashpassword <password>")
		os.Exit(1)
	}

	hash, err := auth.HashPassword(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println(hash)
}
