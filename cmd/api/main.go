package main

import (
	"fmt"
	"log"

	"github.com/pardnchiu/go-faas/internal"
)

func main() {
	err := internal.InitRouter()
	if err != nil {
		log.Fatal(fmt.Printf("Failed to start server: %v\n", err))
	}
}
