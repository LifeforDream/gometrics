package main

import (
	"log"
	"os"
)

func helper() {
	log.Fatal("fatal in helper") // want "found log.Fatal usage outside main.main"
}

// helperNested проверяет, что вызовы, вложенные внутрь блока, тоже
// репортятся, а не только top-level инструкции тела функции.
func helperNested(flag bool) {
	if flag {
		log.Fatal("nested fatal in helper") // want "found log.Fatal usage outside main.main"
	} else {
		os.Exit(1) // want "found os.Exit usage outside main.main"
	}
}

func mainPanic() {
	panic("boom in helper") // want "panic usage"
}

// main - это func main пакета main, поэтому вызовы log.Fatal/os.Exit
// в любом месте её тела, включая вложенные блоки, исключены из проверки.
// У panic() же исключения нет нигде.
func main() {
	helper()
	helperNested(true)
	mainPanic()
	log.Fatal("fatal in main, exempted")
	os.Exit(1)
	if true {
		log.Fatal("nested fatal in main, still exempted")
		os.Exit(1)
	}
	panic("panic even in main is flagged") // want "panic usage"
}
