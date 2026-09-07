// Package main является точкой входа в запуск линтера
package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/LifeforDream/gometrics/internal/exitanalyzer"
)

func main() { singlechecker.Main(exitanalyzer.ExitAnalyzer) }
