package main

import (
	"flag"
)

// flagHTTPAddr holds the HTTP listen address.
var flagHTTPAddr string

// flagLogLevel holds the server log level.
var flagLogLevel string

// flagStoreInterval is the interval in seconds between periodic metric saves to disk; 0 enables synchronous writes.
var flagStoreInterval int

// flagFileStoragePath is the file path where metric values are persisted.
var flagFileStoragePath string

// flagRestore determines whether previously saved metrics are loaded from file on server startup.
var flagRestore bool

// flagDatabaseDSN holds the PostgreSQL connection string (DSN).
var flagDatabaseDSN string

// flagMigrationPath is the directory containing database migration files.
var flagMigrationPath string

// parseFlags registers and parses command-line flags.
func parseFlags() {
	flag.StringVar(&flagHTTPAddr, "a", "localhost:8080", "HTTP server endpoint address")
	flag.StringVar(&flagLogLevel, "l", "Info", "HTTP server log level")
	flag.IntVar(&flagStoreInterval, "i", 300, "interval in seconds between periodic saves to disk; 0 makes writes synchronous")
	flag.StringVar(&flagFileStoragePath, "f", "", "file path for persisting current metric values")
	flag.BoolVar(&flagRestore, "r", true, "load previously saved metric values from file on startup")
	flag.StringVar(&flagDatabaseDSN, "d", "", "Database connection DSN")
	flag.StringVar(&flagMigrationPath, "m", "migrations", "directory containing database migration files")
	flag.Parse()
}
