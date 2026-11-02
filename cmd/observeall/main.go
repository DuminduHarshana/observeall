package main

import (
	"os"

	"github.com/observeall/observeall/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}

