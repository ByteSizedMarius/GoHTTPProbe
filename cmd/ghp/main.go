package main

import (
	"fmt"
	"os"

	"github.com/byte/gohttpprobe/internal/app"
)

func main() {
	if err := app.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
