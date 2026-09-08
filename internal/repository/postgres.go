package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/nikitaw13/metricscollector/internal/model"
	"go.uber.org/zap"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	gaugeTableName   = "gauges"
	counterTableName = "counters"
)

type gaugeRow struct {
	name  string
	value float64
}

type counterRow struct {
	name  string
	delta int64
}

// PostgresStorage wraps a sql.DB connection for PostgreSQL operations.
type PostgresStorage struct {
	db       *sql.DB
	timeouts []time.Duration
	logger   *zap.Logger
}

// Close closes the underlying database connection.
func (ps *PostgresStorage) Close() error {
	return ps.db.Close()
}

var (
	addCounterExec = fmt.Sprintf(`
    INSERT INTO %s (name, delta)
    VALUES ($1, $2)
    ON CONFLICT (name) DO UPDATE
    SET delta = %s.delta + EXCLUDED.delta
	RETURNING delta;`, counterTableName, counterTableName)

	setGaugeExec = fmt.Sprintf(`
	INSERT INTO %s (name, value) 
	VALUES ($1, $2) 
	ON CONFLICT (name) DO UPDATE 
	SET value = $2;`, gaugeTableName)

	getCounterQuery = fmt.Sprintf(`
	SELECT delta 
	FROM %s
	WHERE NAME = $1;`, counterTableName)

	getGaugeQuery = fmt.Sprintf(`
	SELECT value 
	FROM %s
	WHERE NAME = $1;`, gaugeTableName)

	getAllCountersQuery = fmt.Sprintf(`
	SELECT name, delta 
	FROM %s;`, counterTableName)

	getAllGaugesQuery = fmt.Sprintf(`
	SELECT name, value 
	FROM %s;`, gaugeTableName)
)

// NewPostgresStorage creates a PostgresStorage backed by db, with retry backoff timeouts and the provided logger.
func NewPostgresStorage(db *sql.DB, timeouts []time.Duration, logger *zap.Logger) *PostgresStorage {
	return &PostgresStorage{
		db:       db,
		timeouts: timeouts,
		logger:   logger,
	}
}

// NewPostgresStorageFromDSN opens a PostgreSQL connection using the given DSN, runs migrations with progress logged via the provided logger, and returns a PostgresStorage.
func NewPostgresStorageFromDSN(dsn, migrationsPath string, timeouts []time.Duration, logger *zap.Logger) (*PostgresStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open standard sql DB: %w", err)
	}

	driver, err := pgx.WithInstance(db, &pgx.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create migration driver: %w", err)
	}

	migrator, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"pgx",
		driver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize migrator: %w", err)
	}

	logger.Info("applying migrations")
	if err := migrator.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Info("no new migrations to apply")
		} else {
			return nil, fmt.Errorf("migration failed: %w", err)
		}
	} else {
		logger.Info("migrations applied successfully")
	}

	postgresStorage := NewPostgresStorage(db, timeouts, logger)
	return postgresStorage, nil
}

// PingContext verifies the database connection is alive using the provided context.
func (ps *PostgresStorage) PingContext(ctx context.Context) error {
	return ps.db.PingContext(ctx)
}

// SetGauge sets the named gauge metric to the specified value, overwriting any previous value.
func (ps *PostgresStorage) SetGauge(name string, value float64) error {
	return ps.withRetries(func() error {
		return ps.setGauge(name, value)
	})
}

// setGauge writes the gauge value to the database in a single attempt.
func (ps *PostgresStorage) setGauge(name string, value float64) error {
	_, err := ps.db.Exec(setGaugeExec, name, value)

	if err != nil {
		return fmt.Errorf("error writing gauge: %w", err)
	}

	return nil
}

// AddCounter increments the named counter metric by the specified delta.
func (ps *PostgresStorage) AddCounter(name string, delta int64) (int64, error) {
	var newDelta int64

	err := ps.withRetries(func() error {
		var err error
		newDelta, err = ps.addCounter(name, delta)
		return err
	})

	if err != nil {
		return 0, err
	}

	return newDelta, nil
}

// addCounter increments the counter in the database in a single attempt.
func (ps *PostgresStorage) addCounter(name string, delta int64) (int64, error) {
	var newDelta int64
	err := ps.db.QueryRow(addCounterExec, name, delta).Scan(&newDelta)

	if err != nil {
		return 0, fmt.Errorf("error writing counter: %w", err)
	}
	return newDelta, nil
}

// GetGauge returns the value of the named gauge metric.
// Returns an error if the metric does not exist.
func (ps *PostgresStorage) GetGauge(name string) (float64, error) {
	var value float64

	err := ps.withRetries(func() error {
		var err error
		value, err = ps.getGauge(name)
		return err
	})

	if err != nil {
		return 0, err
	}

	return value, nil
}

// getGauge reads the gauge value from the database in a single attempt.
func (ps *PostgresStorage) getGauge(name string) (float64, error) {
	row := ps.db.QueryRow(getGaugeQuery, name)
	var result gaugeRow
	err := row.Scan(&result.value)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("gauge %s %w", name, model.ErrMetricNotFound)
	}

	if err != nil {
		return 0, fmt.Errorf("error scanning row: %w", err)
	}

	return result.value, nil
}

// GetCounter returns the value of the named counter metric.
// Returns an error if the metric does not exist.
func (ps *PostgresStorage) GetCounter(name string) (int64, error) {
	var delta int64

	err := ps.withRetries(func() error {
		var err error
		delta, err = ps.getCounter(name)
		return err
	})

	if err != nil {
		return 0, err
	}

	return delta, nil
}

// getCounter reads the counter delta from the database in a single attempt.
func (ps *PostgresStorage) getCounter(name string) (int64, error) {
	row := ps.db.QueryRow(getCounterQuery, name)
	var result counterRow
	err := row.Scan(&result.delta)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("counter %s %w", name, model.ErrMetricNotFound)
	}

	if err != nil {
		return 0, fmt.Errorf("error scanning row: %w", err)
	}

	return result.delta, nil
}

// GetAllGauges returns a shallow copy of all gauge metrics to prevent external mutation.
func (ps *PostgresStorage) GetAllGauges() (map[string]float64, error) {
	var gauges map[string]float64

	err := ps.withRetries(func() error {
		var err error
		gauges, err = ps.getAllGauges()
		return err
	})

	if err != nil {
		return nil, err
	}

	return gauges, nil
}

// getAllGauges reads all gauges from the database in a single attempt.
func (ps *PostgresStorage) getAllGauges() (map[string]float64, error) {
	rows, err := ps.db.Query(getAllGaugesQuery)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}
	defer rows.Close()

	gauges := make(map[string]float64)

	for rows.Next() {
		var g gaugeRow
		err = rows.Scan(&g.name, &g.value)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}

		gauges[g.name] = g.value
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error encountered during iteration: %w", err)
	}

	return gauges, nil
}

// GetAllCounters returns a shallow copy of all counter metrics to prevent external mutation.
func (ps *PostgresStorage) GetAllCounters() (map[string]int64, error) {
	var counters map[string]int64

	err := ps.withRetries(func() error {
		var err error
		counters, err = ps.getAllCounters()
		return err
	})

	if err != nil {
		return nil, err
	}

	return counters, nil
}

// getAllCounters reads all counters from the database in a single attempt.
func (ps *PostgresStorage) getAllCounters() (map[string]int64, error) {
	rows, err := ps.db.Query(getAllCountersQuery)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}
	defer rows.Close()

	counters := make(map[string]int64)

	for rows.Next() {
		var c counterRow
		err = rows.Scan(&c.name, &c.delta)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}

		counters[c.name] = c.delta
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error encountered during iteration: %w", err)
	}

	return counters, nil
}

// UpdateMetrics applies a batch of metric updates in a single database transaction, retrying on transient failures.
func (ps *PostgresStorage) UpdateMetrics(ctx context.Context, metrics []model.Metric) error {
	return ps.withRetries(
		func() error {
			return ps.updateMetricsTx(ctx, metrics)
		},
	)
}

// updateMetricsTx applies all metric updates within a single database transaction.
func (ps *PostgresStorage) updateMetricsTx(ctx context.Context, metrics []model.Metric) error {
	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning a transaction: %w", err)
	}
	defer tx.Rollback()

	for _, metric := range metrics {
		switch metric.Type {
		case model.Counter:
			_, err = tx.ExecContext(ctx, addCounterExec, metric.ID, *metric.Delta)
			if err != nil {
				return fmt.Errorf("error executing addCounter: %w", err)
			}
		case model.Gauge:
			_, err = tx.ExecContext(ctx, setGaugeExec, metric.ID, *metric.Value)
			if err != nil {
				return fmt.Errorf("error executing setGauge: %w", err)
			}
		default:
			ps.logger.Debug("unknown metric type",
				zap.String("metric_type", metric.Type),
			)
		}

	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error committing a transaction: %w", err)
	}
	return nil
}

// withRetries runs the given operation and retries retriable failures with the configured backoff.
func (ps *PostgresStorage) withRetries(operation func() error) error {
	var lastErr error
	classifier := NewPostgresErrorClassifier()

	for attempt := 0; attempt <= len(ps.timeouts); attempt++ {
		if attempt > 0 {
			time.Sleep(ps.timeouts[attempt-1])
		}
		err := operation()

		if err == nil {
			return nil
		}

		classification := classifier.Classify(err)
		if classification == NonRetriable {
			return err
		}

		lastErr = err
		ps.logger.Debug("retry attempt failed",
			zap.Int("attempt", attempt+1),
			zap.Int("total_attempts", len(ps.timeouts)+1),
			zap.Error(err),
		)
	}
	return fmt.Errorf("operation aborted after %d attempts: %w", len(ps.timeouts)+1, lastErr)
}
