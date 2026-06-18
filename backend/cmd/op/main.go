package main

import (
	"fmt"
	"os"

	"github.com/open-panel/open-panel/internal/cli"
)

func main() {
	if len(os.Args) > 1 {
		if err := cli.RunCommand(os.Args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if err := cli.RunMenu(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
