package main

import (
	"flag"
	"os"

	"github.com/caarlos0/env/v6"
)

type ServerOptions struct {
	RunAddr  string `env:"ADDRESS"`
	LogLevel string `env:"LOG_LEVEL"`
}

func parseOptions(args ...string) (*ServerOptions, error) {
	var serverOptions ServerOptions
	fs := flag.NewFlagSet("server", flag.ContinueOnError)

	fs.StringVar(&serverOptions.RunAddr, "a", "localhost:8080", "address and port to run server")
	fs.StringVar(&serverOptions.LogLevel, "l", "info", "log level")

	if args == nil {
		args = os.Args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	err := env.Parse(&serverOptions)
	if err != nil {
		return nil, err
	}
	return &serverOptions, nil
}
