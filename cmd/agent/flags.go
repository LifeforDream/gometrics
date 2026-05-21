package main

import "flag"

var agentOptions struct {
	address        string
	schema         string
	pollInterval   int
	reportInterval int
}

func parseFlags() {
	flag.StringVar(&agentOptions.address, "a", "localhost:8080", "server address")
	flag.StringVar(&agentOptions.schema, "s", "http://", "protocol schema")
	flag.IntVar(&agentOptions.pollInterval, "p", 2, "poll interval in seconds")
	flag.IntVar(&agentOptions.reportInterval, "r", 10, "report interval in seconds")

	flag.Parse()
}
