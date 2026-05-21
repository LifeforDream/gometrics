package main

import "flag"

var agentOptions struct {
	address        string
	pollInterval   int
	reportInterval int
}

func parseFlags() {
	flag.StringVar(&agentOptions.address, "a", "http://localhost:8080", "server address")
	flag.IntVar(&agentOptions.pollInterval, "p", 2, "poll interval in seconds")
	flag.IntVar(&agentOptions.reportInterval, "r", 10, "report interval in seconds")

	flag.Parse()
}
