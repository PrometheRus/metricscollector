package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/nikitaw13/metricscollector/internal/handler"
	"github.com/nikitaw13/metricscollector/internal/repository"
	"go.uber.org/zap"
)

func main() {
	parseFlags()
	parseEnvs()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := handler.InitLogger(flagLogLevel); err != nil {
		return err
	}

	var storageToUse handler.Repository
	var dbToUse handler.DBPinger
	var timeouts = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

	isSyncWrite := flagStoreInterval == 0

	switch {
	// Database storage
	case flagDatabaseDSN != "":
		ps, err := repository.NewPostgresStorageFromDSN(flagDatabaseDSN, flagMigrationPath, timeouts, handler.Logger)

		if err != nil {
			handler.Logger.Error("failed to initialize postgres storage", zap.Error(err))
			return err
		}
		defer ps.Close()

		storageToUse = ps
		dbToUse = ps

	// Persistent storage
	case flagFileStoragePath != "":
		memStorage := repository.NewMemStorage()
		persistentMemStorage := repository.NewPersistentMemStorage(memStorage, flagFileStoragePath, isSyncWrite)

		if flagRestore {
			err := persistentMemStorage.Restore()

			if errors.Is(err, os.ErrNotExist) {
				handler.Logger.Warn("file does not exist", zap.Error(err))
			} else if err != nil {
				handler.Logger.Error("failed to restore metrics from file", zap.Error(err))
			}
		}

		storageToUse = persistentMemStorage
		dbToUse = persistentMemStorage

	// In-memory storage
	default:
		memStorage := repository.NewMemStorage()
		storageToUse = memStorage
		dbToUse = memStorage
	}

	metricsHandler := handler.NewMetricsHandler(storageToUse, dbToUse)

	router := metricsHandler.NewRouter()

	srv := &http.Server{
		Addr:         flagHTTPAddr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      router,
	}

	if persistentStorage, ok := storageToUse.(*repository.PersistentMemStorage); ok {
		if !isSyncWrite {
			go persistentStorage.PeriodicSave(time.Duration(flagStoreInterval) * time.Second)
		}
	}

	handler.Logger.Info("server is running", zap.String("address", flagHTTPAddr), zap.String("log_level", flagLogLevel))
	return srv.ListenAndServe()
}
