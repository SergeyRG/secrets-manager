package psql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"iter"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	basePSQLInfra "github.com/SergeyRG/secrets-manager/server/internal/shared/infrastructure/psql"
	"github.com/jackc/pgx/v5/pgconn"
)

const errPSQLCodeUniqueViolation = "23505"

var (
	ErrSecretsMetadataConflict = errors.New("Секрет с данным именем и версией уже существует")
)

type SecretsMetadataRepo struct {
	basePSQLInfra.BaseRepository
}

func NewSecretsMetadataRepo(txm *basePSQLInfra.PSQLTxManager) *SecretsMetadataRepo {
	return &SecretsMetadataRepo{
		BaseRepository: basePSQLInfra.BaseRepository{TxManager: txm},
	}
}

func (repo *SecretsMetadataRepo) AddSecretMetadata(ctx context.Context, sm secretsDomain.SecretsMetadata) error {
	qe := repo.GetExecutor(ctx)
	query := `INSERT INTO secrets_metadata 
				(user_id, secret_id, secret_name, secret_type, time_creation, version, version_id)
			VALUES
				($1, $2, $3 , $4, $5, $6, $7)`

	_, err := qe.ExecContext(
		ctx,
		query,
		sm.UserID,
		sm.SecretID,
		sm.SecretName,
		sm.SecretType.ToString(),
		sm.TimeCreation,
		sm.Version,
		sm.VersionID,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == errPSQLCodeUniqueViolation && pgErr.ConstraintName == "uq_user_secret_name" {
				return ErrSecretsMetadataConflict
			}
		}

		return fmt.Errorf("непредвиденная ошибка выполнения запроса: %w", err)
	}

	return nil
}

func (repo *SecretsMetadataRepo) GetUserSecretMetadataByName(
	ctx context.Context,
	uID domain.UserID,
	sName string,
	version int,
) (secretsDomain.SecretsMetadata, error) {
	qe := repo.GetExecutor(ctx)
	var (
		query  string
		result *sql.Row
	)
	if version == 0 {
		query = `SELECT FOR UPDATE
 				user_id, secret_id, secret_name, secret_type, time_creation, version, version_id
 			  FROM
 				secrets_metadata
 			  WHERE
			  	user_id = $1 and secret_name = $2 
			  ORDER BY version DESC
 			  LIMIT 1;`
		result = qe.QueryRowContext(ctx, query, string(uID), sName)
	} else {
		query = `SELECT FOR UPDATE
 				user_id, secret_id, secret_name, secret_type, time_creation, version, version_id
 			  FROM
 				secrets_metadata
 			  WHERE
			  	user_id = $1 and secret_name = $2 and version = $3`
		result = qe.QueryRowContext(ctx, query, string(uID), sName, version)
	}
	sm := secretsDomain.SecretsMetadata{}
	secretType := ""

	err := result.Scan(&sm.UserID, &sm.SecretID, &sm.SecretName, &secretType, &sm.TimeCreation, &sm.Version, &sm.VersionID)
	if err != nil {
		return secretsDomain.SecretsMetadata{}, fmt.Errorf("ошибка получения данных из БД: %w", err)
	}
	sm.SecretType = secretsDomain.SecretTypeFromString(secretType)

	return sm, nil
}

func (repo *SecretsMetadataRepo) GetUserSecretsMetadataPage(
	ctx context.Context,
	uID domain.UserID,
	page int,
	perPage int,
) iter.Seq2[secretsDomain.SecretsMetadata, error] {
	return func(yield func(secretsDomain.SecretsMetadata, error) bool) {
		qe := repo.GetExecutor(ctx)
		query := `
				SELECT
    				user_id, secret_id, secret_name, secret_type, time_creation, version, version_id
				FROM
    				secrets_metadata
				WHERE
    				user_id = $1
    			AND 
					(secret_id, version) IN (
				        SELECT 
							secret_id, MAX(version)
        				FROM 
							secrets_metadata
        				WHERE
							user_id = $1
        				GROUP BY 
							secret_id
    					)
				ORDER BY
    				secret_name
				LIMIT $2
			OFFSET $3;`

		rows, err := qe.QueryContext(ctx, query, string(uID), perPage, (page-1)*perPage)

		if err != nil {
			yield(secretsDomain.SecretsMetadata{}, fmt.Errorf("ошибка получения данных из БД: %w", err))
			return
		}
		defer rows.Close()

		for rows.Next() {
			sm := secretsDomain.SecretsMetadata{}
			secretType := ""
			err := rows.Scan(
				&sm.UserID,
				&sm.SecretID,
				&sm.SecretName,
				&secretType,
				&sm.TimeCreation,
				&sm.Version,
				&sm.VersionID,
			)
			if err != nil {
				yield(secretsDomain.SecretsMetadata{}, fmt.Errorf("ошибка получения данных из БД: %w", err))
				return
			}
			sm.SecretType = secretsDomain.SecretTypeFromString(secretType)
			if !yield(sm, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(secretsDomain.SecretsMetadata{}, fmt.Errorf("ошибка при чтении строк БД: %w", err))
		}
	}
}
