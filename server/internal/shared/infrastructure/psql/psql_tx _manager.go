package psql

import (
	"context"
	"database/sql"
	"fmt"

	usecases "github.com/SergeyRG/secrets-manager/server/internal/shared/use-cases"
)

type txCtxKeyType struct{}

var txKey = txCtxKeyType{}

type PSQLTxManager struct {
	db *sql.DB
}

func NewTxManager(db *sql.DB) *PSQLTxManager {
	return &PSQLTxManager{db: db}
}

func (m *PSQLTxManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ошибка создания транзакции СУБД: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	txCtx := context.WithValue(ctx, txKey, tx)
	err = fn(txCtx)

	if err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("shared tx commit failed: %w", err)
	}

	return nil
}

func (m *PSQLTxManager) GetExecutor(ctx context.Context) usecases.QueryExecutor {
	if tx, ok := ctx.Value(txKey).(*sql.Tx); ok && tx != nil {
		return tx
	}
	return m.db
}
