package psql_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	"github.com/SergeyRG/secrets-manager/server/internal/secrets/infrastructure/db/psql"
	basePSQLInfra "github.com/SergeyRG/secrets-manager/server/internal/shared/infrastructure/psql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestSecretsTextdataRepo_AddSecretData(t *testing.T) {
	sm := secretsDomain.SecretsMetadata{
		VersionID: domain.SecretVersionID("version-uuid-111"),
	}
	testData := []byte("confidential-free-text-payload")
	insertQueryRegex := `(?i)^[\s]*INSERT\s+INTO\s+secrets_text_data[\s\S]*`

	t.Run("Успешное добавление текстовых данных", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		txm := basePSQLInfra.NewTxManager(db)
		repo := psql.NewSecretsTextdataRepo(txm)

		mock.ExpectExec(insertQueryRegex).
			WithArgs(sm.VersionID, string(testData)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.AddSecretData(context.Background(), sm, testData)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Ошибка СУБД при сохранении данных", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		txm := basePSQLInfra.NewTxManager(db)
		repo := psql.NewSecretsTextdataRepo(txm)

		mock.ExpectExec(insertQueryRegex).
			WithArgs(gomockAnyArg(), gomockAnyArg()).
			WillReturnError(errors.New("postgres disk i/o error"))

		err = repo.AddSecretData(context.Background(), sm, testData)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка сохранения данных в БД")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSecretsTextdataRepo_GetSecretData(t *testing.T) {
	sm := secretsDomain.SecretsMetadata{
		VersionID: domain.SecretVersionID("version-uuid-222"),
	}
	testDataStr := "my-stored-text-secret-data"
	selectQueryRegex := `(?i)^[\s]*SELECT\s+secret_data\s+FROM\s+secrets_text_data[\s\S]*`

	t.Run("Успешное получение текстовых данных", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		txm := basePSQLInfra.NewTxManager(db)
		repo := psql.NewSecretsTextdataRepo(txm)

		rows := sqlmock.NewRows([]string{"secret_data"}).AddRow(testDataStr)
		mock.ExpectQuery(selectQueryRegex).
			WithArgs(sm.VersionID).
			WillReturnRows(rows)

		resBytes, err := repo.GetSecretData(context.Background(), sm)

		require.NoError(t, err)
		assert.Equal(t, []byte(testDataStr), resBytes)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Ошибка: Данные версии секрета не найдены в БД (sql.ErrNoRows)", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		txm := basePSQLInfra.NewTxManager(db)
		repo := psql.NewSecretsTextdataRepo(txm)

		mock.ExpectQuery(selectQueryRegex).
			WithArgs(sm.VersionID).
			WillReturnError(sql.ErrNoRows)

		resBytes, err := repo.GetSecretData(context.Background(), sm)

		assert.Nil(t, resBytes)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка получения данных в БД")
		assert.True(t, errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "sql: no rows"))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func gomockAnyArg() sqlmock.Argument {
	return sqlmock.AnyArg()
}
