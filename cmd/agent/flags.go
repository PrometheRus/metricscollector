package main

import (
	"flag"
)

// flagServerAddr holds the HTTP address of the metrics server (-a).
var flagServerAddr string

// flagReportInterval holds the interval in seconds between metric reports (-r).
var flagReportInterval int

// flagPollInterval holds the interval in seconds between metric collection cycles (-p).
var flagPollInterval int

// parseFlags registers and parses command-line flags.
func parseFlags() {
	flag.StringVar(&flagServerAddr, "a", "localhost:8080", "HTTP server endpoint address")
	flag.IntVar(&flagReportInterval, "r", 10, "Metrics report interval in seconds")
	flag.IntVar(&flagPollInterval, "p", 2, "Metrics poll interval in seconds")
	flag.Parse()
}
