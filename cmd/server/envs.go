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
		flagHTTPAddr = envAddress
	}

	envLogLevel, found := os.LookupEnv("LOG_LEVEL")
	if found {
		flagLogLevel = envLogLevel
	}

	envStoreInterval, found := os.LookupEnv("STORE_INTERVAL")
	if found {
		intervalSec, err := strconv.Atoi(envStoreInterval)

		if err != nil {
			log.Fatal("failed to parse STORE_INTERVAL")
		}
		flagStoreInterval = intervalSec
	}

	envFileStoragePath, found := os.LookupEnv("FILE_STORAGE_PATH")
	if found {
		flagFileStoragePath = envFileStoragePath
	}

	envRestore, found := os.LookupEnv("RESTORE")
	if found {
		shouldRestore, err := strconv.ParseBool(envRestore)

		if err != nil {
			log.Fatal("failed to parse RESTORE")
		}
		flagRestore = shouldRestore
	}

	envDatabaseDSN, found := os.LookupEnv("DATABASE_DSN")
	if found {
		flagDatabaseDSN = envDatabaseDSN
	}

	envMigrationPath, found := os.LookupEnv("MIGRATION_PATH")
	if found {
		flagMigrationPath = envMigrationPath
	}
}
