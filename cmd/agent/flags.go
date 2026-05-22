package main

import (
	"flag"
	"strings"
)

var agentOptions struct {
	address        string
	secure         bool
	pollInterval   int
	reportInterval int
}

func parseFlags() {
	flag.StringVar(&agentOptions.address, "a", "localhost:8080", "server address")
	flag.BoolVar(&agentOptions.secure, "secure", false, "flag to indicate usage of secure channel")
	flag.IntVar(&agentOptions.pollInterval, "p", 2, "poll interval in seconds")
	flag.IntVar(&agentOptions.reportInterval, "r", 10, "report interval in seconds")

	flag.Parse()
}

func constructAddress() string {
	var serverAddr string
	// since -a may be without scheme
	// we explicitly check for scheme
	// also url.Parse() does not detect missing scheme
	address := strings.ToLower(agentOptions.address)
	if strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://") {
		serverAddr = agentOptions.address
	} else {
		if agentOptions.secure {
			serverAddr = "https://" + agentOptions.address
		} else {
			serverAddr = "http://" + agentOptions.address
		}
	}
	return serverAddr
}
