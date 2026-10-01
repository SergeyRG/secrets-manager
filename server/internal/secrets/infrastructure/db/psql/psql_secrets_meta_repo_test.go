package psql_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	"github.com/SergeyRG/secrets-manager/server/internal/secrets/infrastructure/db/psql"
	basePSQLInfra "github.com/SergeyRG/secrets-manager/server/internal/shared/infrastructure/psql"
)

func TestSecretsMetadataRepo_AddSecretMetadata(t *testing.T) {
	sm := secretsDomain.SecretsMetadata{
		UserID:       domain.UserID("user-1"),
		SecretID:     domain.SecretID("secret-uuid"),
		SecretName:   "my_bank_card",
		SecretType:   secretsDomain.SecretTypeBankCard,
		TimeCreation: time.Now().UTC(),
		Version:      1,
		VersionID:    domain.SecretVersionID("version-uuid"),
	}

	queryRegex := `(?i)^[\s]*INSERT\s+INTO\s+secrets_metadata[\s\S]*`

	t.Run("Успешное добавление метаданных", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		txm := basePSQLInfra.NewTxManager(db)
		repo := psql.NewSecretsMetadataRepo(txm)

		mock.ExpectExec(queryRegex).
			WithArgs(sm.UserID, sm.SecretID, sm.SecretName, sm.SecretType.ToString(), sm.TimeCreation, sm.Version, sm.VersionID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.AddSecretMetadata(context.Background(), sm)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Ошибка: Секрет с таким именем и версией уже существует (Unique Violation)", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		txm := basePSQLInfra.NewTxManager(db)
		repo := psql.NewSecretsMetadataRepo(txm)

		// Имитируем ошибку нарушения уникальности pq/pgx драйвера
		pgErr := &pgconn.PgError{
			Code:           "23505",
			ConstraintName: "uq_user_secret_name",
		}

		mock.ExpectExec(queryRegex).
			WithArgs(gomockAny(), gomockAny(), gomockAny(), gomockAny(), gomockAny(), gomockAny(), gomockAny()).
			WillReturnError(pgErr)

		err = repo.AddSecretMetadata(context.Background(), sm)
		assert.ErrorIs(t, err, psql.ErrSecretsMetadataConflict)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSecretsMetadataRepo_GetUserSecretMetadataByName(t *testing.T) {
	uID := domain.UserID("user-1")
	secretName := "my_text"
	timeNow := time.Now().UTC()

	columns := []string{"user_id", "secret_id", "secret_name", "secret_type", "time_creation", "version", "version_id"}

	t.Run("Успешное получение последней версии (version == 0)", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		txm := basePSQLInfra.NewTxManager(db)
		repo := psql.NewSecretsMetadataRepo(txm)

		rows := sqlmock.NewRows(columns).
			AddRow(string(uID), "secret-uuid", secretName, "SECRET_TYPE_FREE_TEXT", timeNow, int64(5), "version-uuid-5")

		// Проверяем, что SQL содержит ORDER BY version DESC для кейса с нулевой версией
		mock.ExpectQuery(`(?i)ORDER\s+BY\s+version\s+DESC`).
			WithArgs(string(uID), secretName).
			WillReturnRows(rows)

		res, err := repo.GetUserSecretMetadataByName(context.Background(), uID, secretName, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(5), res.Version)
		assert.Equal(t, secretsDomain.SecretTypeFreeText, res.SecretType)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Успешное получение конкретной версии (version > 0)", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		txm := basePSQLInfra.NewTxManager(db)
		repo := psql.NewSecretsMetadataRepo(txm)

		rows := sqlmock.NewRows(columns).
			AddRow(string(uID), "secret-uuid", secretName, "SECRET_TYPE_FREE_TEXT", timeNow, int64(2), "version-uuid-2")

		// Проверяем, что в аргументах передается точная версия (2)
		mock.ExpectQuery(`(?i)version\s*=\s*\$3`).
			WithArgs(string(uID), secretName, 2).
			WillReturnRows(rows)

		res, err := repo.GetUserSecretMetadataByName(context.Background(), uID, secretName, 2)
		require.NoError(t, err)
		assert.Equal(t, int64(2), res.Version)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSecretsMetadataRepo_GetUserSecretsMetadataPage(t *testing.T) {
	uID := domain.UserID("user-1")
	columns := []string{"user_id", "secret_id", "secret_name", "secret_type", "time_creation", "version", "version_id"}
	timeNow := time.Now().UTC()

	t.Run("Успешный проход по итератору страниц", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		txm := basePSQLInfra.NewTxManager(db)
		repo := psql.NewSecretsMetadataRepo(txm)

		rows := sqlmock.NewRows(columns).
			AddRow(string(uID), "id-1", "auth_data", "SECRET_TYPE_AUTH_DATA", timeNow, int64(1), "v-1").
			AddRow(string(uID), "id-2", "bank_card", "SECRET_TYPE_BANK_CARD", timeNow, int64(3), "v-3")

		mock.ExpectQuery(`(?i)SELECT[\s\S]*FROM\s+secrets_metadata`).
			WithArgs(string(uID), 20, 0). // perPage=20, offset=(1-1)*20 = 0
			WillReturnRows(rows)

		// Получаем итератор
		seq := repo.GetUserSecretsMetadataPage(context.Background(), uID, 1, 20)

		// Проходим по циклу
		var collected []secretsDomain.SecretsMetadata
		for sm, err := range seq {
			require.NoError(t, err)
			collected = append(collected, sm)
		}

		assert.Len(t, collected, 2)
		assert.Equal(t, "auth_data", collected[0].SecretName)
		assert.Equal(t, "bank_card", collected[1].SecretName)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Ошибка СУБД внутри итератора", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		txm := basePSQLInfra.NewTxManager(db)
		repo := psql.NewSecretsMetadataRepo(txm)

		mock.ExpectQuery(`(?i)SELECT[\s\S]*FROM\s+secrets_metadata`).
			WithArgs(string(uID), 10, 10).
			WillReturnError(errors.New("postgres connection reset"))

		seq := repo.GetUserSecretsMetadataPage(context.Background(), uID, 2, 10)

		// Проверяем, что ошибка корректно пробрасывается наружу через итератор
		didSeeError := false
		for _, err := range seq {
			if err != nil {
				didSeeError = true
				assert.Contains(t, err.Error(), "ошибка получения данных из БД")
			}
		}

		assert.True(t, didSeeError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func gomockAny() sqlmock.Argument {
	return sqlmock.AnyArg()
}
