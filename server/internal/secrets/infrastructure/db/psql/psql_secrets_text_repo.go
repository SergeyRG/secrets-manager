package psql

import (
	"context"
	"fmt"

	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	basePSQLInfra "github.com/SergeyRG/secrets-manager/server/internal/shared/infrastructure/psql"
)

type SecretsTextdataRepo struct {
	basePSQLInfra.BaseRepository
}

func NewSecretsTextdataRepo(txm *basePSQLInfra.PSQLTxManager) *SecretsTextdataRepo {
	return &SecretsTextdataRepo{
		BaseRepository: basePSQLInfra.BaseRepository{TxManager: txm},
	}
}

func (repo *SecretsTextdataRepo) AddSecretData(ctx context.Context, sm secretsDomain.SecretsMetadata, data []byte) error {
	qe := repo.GetExecutor(ctx)
	query := `INSERT INTO secrets_text_data 
				(secret_version_id, secret_data)
			VALUES
				($1, $2)`

	_, err := qe.ExecContext(
		ctx,
		query,
		sm.VersionID,
		string(data),
	)

	if err != nil {
		return fmt.Errorf("ошибка сохранения данных в БД: %w", err)
	}

	return nil
}

func (repo *SecretsTextdataRepo) GetSecretData(ctx context.Context, sm secretsDomain.SecretsMetadata) ([]byte, error) {
	qe := repo.GetExecutor(ctx)
	query := `SELECT 
				secret_data 
			FROM
			    secrets_text_data 
			WHERE
				secret_version_id = $1`

	result := qe.QueryRowContext(
		ctx,
		query,
		sm.VersionID,
	)

	var data string
	err := result.Scan(&data)

	if err != nil {
		return nil, fmt.Errorf("ошибка получения данных в БД: %w", err)
	}

	return []byte(data), nil
}
