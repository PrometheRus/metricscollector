package main

import (
	"log"
	"os"
	"strconv"
)

// parseEnvs overrides flag variables with values from environment variables.
func parseEnvs() {
	envAddress, found := os.LookupEnv("ADDRESS")
	if found {
		flagServerAddr = envAddress
	}

	envReportInterval, found := os.LookupEnv("REPORT_INTERVAL")
	if found {
		intervalSec, err := strconv.Atoi(envReportInterval)

		if err != nil {
			log.Fatal("failed to parse REPORT_INTERVAL")
		}
		flagReportInterval = intervalSec
	}

	envPollInterval, found := os.LookupEnv("POLL_INTERVAL")
	if found {
		intervalSec, err := strconv.Atoi(envPollInterval)

		if err != nil {
			log.Fatal("failed to parse POLL_INTERVAL")
		}

		flagPollInterval = intervalSec
	}
}
