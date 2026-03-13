// Worker entrypoint
// Known complexity: main=3, processJob=5
package main

import (
	"fmt"
	"os"
)

func main() {
	// complexity: 1 (base) + 2 decision points = 3
	mode := os.Getenv("WORKER_MODE")
	if mode == "" {
		mode = "default"
	}

	if mode == "debug" {
		fmt.Println("Debug mode enabled")
	}

	fmt.Printf("Worker starting in %s mode\n", mode)
}
