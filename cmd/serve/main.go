package main

import (
	"os"

	"github.com/kszab0/serve"
)

func main() {
	os.Exit(serve.CLI(os.Args[1:]))
}
