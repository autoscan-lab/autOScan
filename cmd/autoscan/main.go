package main

import (
	"fmt"
	"os"

	"github.com/autoscan-lab/autoscan/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
