// Package main является точкой входа для запуска генератора
package main

import (
	"log"
	"os"

	"github.com/LifeforDream/gometrics/internal/reset"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	if err := reset.Run(root); err != nil {
		log.Fatal(err)
	}
}
