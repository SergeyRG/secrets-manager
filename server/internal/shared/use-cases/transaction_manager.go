package usecases

import (
	"context"
	"database/sql"
)

type QueryExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	GetExecutor(ctx context.Context) QueryExecutor
}
