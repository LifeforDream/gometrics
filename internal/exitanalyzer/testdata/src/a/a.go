package a

import (
	"fmt"
	"log"
	"os"
)

func doPanic() {
	panic("boom") // want "panic usage"
}

func doPanicNested(flag bool) {
	if flag {
		panic("nested boom") // want "panic usage"
	}
}

func doPanicGo() {
	go panic("boom") // want "panic usage in go statement"
}

func doPanicDefer() {
	defer panic("boom") // want "panic usage in defer statement"
}

func doFatal() {
	log.Fatal("fatal error") // want "found log.Fatal usage outside main.main"
}

func doExit() {
	os.Exit(1) // want "found os.Exit usage outside main.main"
}

// doFatalNested и doExitNested проверяют, что log.Fatal/os.Exit
// обнаруживаются, даже если вызов вложен внутрь if-блока, а не только
// когда он является top-level инструкцией тела функции.
func doFatalNested(flag bool) {
	if flag {
		log.Fatal("nested fatal") // want "found log.Fatal usage outside main.main"
	}
}

func doExitNested(flag bool) {
	if flag {
		os.Exit(1) // want "found os.Exit usage outside main.main"
	}
}

// doFatalln демонстрирует, что сравнение идёт только по точному имени
// функции "Fatal", поэтому log.Fatalln не обнаруживается.
func doFatalln() {
	log.Fatalln("not an exact match for Fatal, not detected")
}

// main объявлена в пакете, который не называется буквально "main",
// поэтому исключение для main.main не действует и вызов всё равно
// репортится.
func main() {
	log.Fatal("fatal in non-main package's main func") // want "found log.Fatal usage outside main.main"
}

func doLog() {
	fmt.Println("just logging, nothing forbidden here")
	log.Println("also fine")
}

// initFatal проверяет, что запрещённый вызов внутри функционального
// литерала тоже обнаруживается, поскольку проверка основана на позиции
// вызова, а не ограничена телами *ast.FuncDecl.
var initFatal = func() {
	log.Fatal("in func literal") // want "found log.Fatal usage outside main.main"
}
