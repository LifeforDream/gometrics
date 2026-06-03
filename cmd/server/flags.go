package main

import (
	"flag"
	"os"

	"github.com/caarlos0/env/v6"
)

type ServerOptions struct {
	RunAddr string `env:"ADDRESS"`
}

func parseOptions(args ...string) (*ServerOptions, error) {
	var serverOptions ServerOptions
	fs := flag.NewFlagSet("server", flag.ExitOnError)

	fs.StringVar(&serverOptions.RunAddr, "a", "localhost:8080", "address and port to run server")

	if args == nil {
		args = os.Args[1:]
	}
	fs.Parse(args)

	err := env.Parse(&serverOptions)
	if err != nil {
		return nil, err
	}
	return &serverOptions, nil
}
