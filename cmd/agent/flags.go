package main

import (
	"flag"
	"os"
	"strings"

	"github.com/caarlos0/env/v6"
)

type AgentOptions struct {
	Address        string `env:"ADDRESS"`
	Secure         bool
	PollInterval   int `env:"POLL_INTERVAL"`
	ReportInterval int `env:"REPORT_INTERVAL"`
}

func parseOptions(args ...string) (*AgentOptions, error) {
	var agentOptions AgentOptions
	fs := flag.NewFlagSet("agent", flag.ExitOnError)

	fs.StringVar(&agentOptions.Address, "a", "localhost:8080", "server address")
	fs.BoolVar(&agentOptions.Secure, "secure", false, "flag to indicate usage of secure channel")
	fs.IntVar(&agentOptions.PollInterval, "p", 2, "poll interval in seconds")
	fs.IntVar(&agentOptions.ReportInterval, "r", 10, "report interval in seconds")

	if args == nil {
		args = os.Args[1:]
	}
	fs.Parse(args)

	err := env.Parse(&agentOptions)
	if err != nil {
		return nil, err
	}

	return &agentOptions, nil
}

func constructAddress(agentOptions *AgentOptions) string {
	var serverAddr string
	// since -a may be without scheme
	// we explicitly check for scheme
	// also url.Parse() does not detect missing scheme
	address := strings.ToLower(agentOptions.Address)
	if strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://") {
		serverAddr = agentOptions.Address
	} else {
		if agentOptions.Secure {
			serverAddr = "https://" + agentOptions.Address
		} else {
			serverAddr = "http://" + agentOptions.Address
		}
	}
	return serverAddr
}
