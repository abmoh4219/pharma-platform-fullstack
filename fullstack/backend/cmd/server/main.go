package main

import (
	"log"

	"pharma-platform/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
