package handler

import "context"

// DBPinger defines the ping method required to check database connectivity from handlers.
type DBPinger interface {
	PingContext(ctx context.Context) error
}
