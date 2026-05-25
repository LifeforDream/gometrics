package main

import "flag"

var serverOptions struct {
	runAddr string
}

// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных
func parseFlags() {
	flag.StringVar(&serverOptions.runAddr, "a", "localhost:8080", "address and port to run server")
	flag.Parse()
}
