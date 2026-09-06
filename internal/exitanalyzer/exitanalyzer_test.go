package exitanalyzer

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestExitAnalyzer прогоняет ExitAnalyzer на тестовых пакетах в testdata:
// пакет "a" (обычный, не main) и пакет "b" (package main), проверяя
// диагностики panic()/log.Fatal()/os.Exit() через "// want"-аннотации.
func TestExitAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, ExitAnalyzer, "a", "b", "c")
}
