package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	cfg, err := loadConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid configuration: %v\n", err)
		os.Exit(2)
	}

	if err := runStress(context.Background(), cfg); err != nil {
		fmt.Fprintf(os.Stderr, "stress test failed: %v\n", err)
		os.Exit(1)
	}
}
