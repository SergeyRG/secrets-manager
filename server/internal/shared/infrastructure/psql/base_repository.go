package psql

import (
	"context"

	usecases "github.com/SergeyRG/secrets-manager/server/internal/shared/use-cases"
)

type BaseRepository struct {
	TxManager usecases.TransactionManager
}

func NewBaseRepo(txm usecases.TransactionManager) *BaseRepository {
	return &BaseRepository{TxManager: txm}
}

func (br *BaseRepository) GetExecutor(ctx context.Context) usecases.QueryExecutor {
	return br.TxManager.GetExecutor(ctx)
}
