package psql

import (
	"context"

	"github.com/SergeyRG/secrets-manager/server/internal/files/domain"
	basePSQLInfra "github.com/SergeyRG/secrets-manager/server/internal/shared/infrastructure/psql"
)

type PSQLFileStateRepository struct {
	basePSQLInfra.BaseRepository
}

func NewPSQLFileStateRepository(txm *basePSQLInfra.PSQLTxManager) *PSQLFileStateRepository {
	return &PSQLFileStateRepository{
		BaseRepository: basePSQLInfra.BaseRepository{TxManager: txm},
	}
}

func (repo *PSQLFileStateRepository) AddFileState(ctx context.Context, fID domain.FileID, fState domain.FileState) error {
	qe := repo.GetExecutor(ctx)
	query := `
			INSERT INTO files_state
				(file_id, state)
			VALUES 
				($1, $2)
			ON CONFLICT 
				(file_id) 
			DO UPDATE SET 
    			state = EXCLUDED.state;`

	_, err := qe.ExecContext(ctx, query, fID, fState.ToString())
	if err != nil {
		return err
	}

	return nil
}

func (repo *PSQLFileStateRepository) GetFileState(ctx context.Context, fID domain.FileID) (domain.FileState, error) {
	return domain.FileState(1), nil
}
