package psql_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	securityDomain "github.com/SergeyRG/secrets-manager/server/internal/security/domain"
	"github.com/SergeyRG/secrets-manager/server/internal/security/infrastructure/db/psql"
	usecases "github.com/SergeyRG/secrets-manager/server/internal/security/use-cases"
	sharedUseCases "github.com/SergeyRG/secrets-manager/server/internal/shared/use-cases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

type TestTxManager struct {
	db *sql.DB
}

func (m TestTxManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (m TestTxManager) GetExecutor(ctx context.Context) sharedUseCases.QueryExecutor {
	return m.db // sql.DB из коробки реализует методы ExecContext, QueryContext и QueryRowContext
}

func TestPSQLEncryptedKeyRepo(t *testing.T) {
	testUserID := domain.UserID("user-db-777")
	testKey := securityDomain.EncryptedKey("my-secure-database-encrypted-key")

	// Регулярные выражения для SQL-запросов (экранируем пробелы и переносы)
	getQueryRegex := `(?i)^[\s]*SELECT\s+encrypted_key\s+FROM\s+encrypted_keys\s+WHERE\s+user_id\s*=\s*\$1`
	setQueryRegex := `(?i)^[\s]*INSERT\s+INTO\s+encrypted_keys[\s\S]*VALUES[\s\S]*ON\s+CONFLICT[\s\S]*`

	t.Run("GetUserEncryptedKey - Успешное чтение ключа", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		txm := TestTxManager{db: db}
		repo := psql.NewPSQLEncryptedKeyRepo(txm)

		rows := sqlmock.NewRows([]string{"encrypted_key"}).AddRow(string(testKey))
		mock.ExpectQuery(getQueryRegex).
			WithArgs(string(testUserID)).
			WillReturnRows(rows)

		resKey, err := repo.GetUserEncryptedKey(context.Background(), testUserID)

		assert.NoError(t, err)
		assert.Equal(t, testKey, resKey)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetUserEncryptedKey - Ключ пользователя отсутствует (404 / sql.ErrNoRows)", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		txm := TestTxManager{db: db}
		repo := psql.NewPSQLEncryptedKeyRepo(txm)

		mock.ExpectQuery(getQueryRegex).
			WithArgs(string(testUserID)).
			WillReturnError(sql.ErrNoRows)

		resKey, err := repo.GetUserEncryptedKey(context.Background(), testUserID)

		assert.Empty(t, resKey)
		assert.ErrorIs(t, err, usecases.ErrEncryptionKeyDoesntExists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetUserEncryptedKey - Внутренняя ошибка СУБД при чтении", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		txm := TestTxManager{db: db}
		repo := psql.NewPSQLEncryptedKeyRepo(txm)

		mock.ExpectQuery(getQueryRegex).
			WithArgs(string(testUserID)).
			WillReturnError(errors.New("fatal postgres hardware error"))

		_, err = repo.GetUserEncryptedKey(context.Background(), testUserID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fatal postgres hardware error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("SetUserEncryptedKey - Успешная вставка/обновление ключа", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		txm := TestTxManager{db: db}
		repo := psql.NewPSQLEncryptedKeyRepo(txm)

		mock.ExpectExec(setQueryRegex).
			WithArgs(string(testUserID), string(testKey)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.SetUserEncryptedKey(context.Background(), testUserID, testKey)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("SetUserEncryptedKey - Ошибка СУБД при записи", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		txm := TestTxManager{db: db}
		repo := psql.NewPSQLEncryptedKeyRepo(txm)

		mock.ExpectExec(setQueryRegex).
			WithArgs(gomockAnyArg(), gomockAnyArg()).
			WillReturnError(errors.New("sql: table encrypted_keys is locked"))

		// Выполнение
		err = repo.SetUserEncryptedKey(context.Background(), testUserID, testKey)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "table encrypted_keys is locked")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func gomockAnyArg() sqlmock.Argument {
	return sqlmock.AnyArg()
}
