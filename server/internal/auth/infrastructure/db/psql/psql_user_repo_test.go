package psql_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	authDomain "github.com/SergeyRG/secrets-manager/server/internal/auth/domain"
	"github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/db/psql"
	usecases "github.com/SergeyRG/secrets-manager/server/internal/auth/use-cases" // Скорректируйте путь импорта под ваш проект
	basePSQLInfra "github.com/SergeyRG/secrets-manager/server/internal/shared/infrastructure/psql"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestPSQLUserRepo_GetUserByLogin(t *testing.T) {
	login := "test_user"
	selectQueryRegex := `(?i)^[\s]*SELECT\s+user_id,\s*login,\s*pwd_hash\s+FROM\s+users\s+WHERE\s+login\s*=\s*\$1`

	t.Run("Успешное получение пользователя", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		txm := basePSQLInfra.NewTxManager(db)
		repo := psql.NewPSQLUserRepo(txm)

		rows := sqlmock.NewRows([]string{"user_id", "login", "pwd_hash"}).
			AddRow("user-uuid-123", login, "hashed_password_string")

		mock.ExpectQuery(selectQueryRegex).
			WithArgs(login).
			WillReturnRows(rows)

		user, err := repo.GetUserByLogin(context.Background(), login)

		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, "user-uuid-123", string(user.UserID))
		assert.Equal(t, login, user.Login)
		assert.Equal(t, "hashed_password_string", user.PwdHash)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Ошибка: Пользователь не найден (sql.ErrNoRows)", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		txm := basePSQLInfra.NewTxManager(db)
		repo := psql.NewPSQLUserRepo(txm)

		mock.ExpectQuery(selectQueryRegex).
			WithArgs(login).
			WillReturnError(sql.ErrNoRows)

		user, err := repo.GetUserByLogin(context.Background(), login)

		assert.Nil(t, user)
		assert.ErrorIs(t, err, usecases.ErrWrongLoginOrPassword)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Непредвиденная ошибка СУБД при чтении", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		txm := basePSQLInfra.NewTxManager(db)
		repo := psql.NewPSQLUserRepo(txm)

		mock.ExpectQuery(selectQueryRegex).
			WithArgs(login).
			WillReturnError(errors.New("connection timeout"))

		user, err := repo.GetUserByLogin(context.Background(), login)

		assert.Nil(t, user)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка получения пользователя из БД")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPSQLUserRepo_AddUser(t *testing.T) {
	user := authDomain.User{
		UserID:  "new-user-uuid",
		Login:   "new_login",
		PwdHash: "secret_hash",
	}
	insertQueryRegex := `(?i)^[\s]*INSERT\s+INTO\s+users[\s\S]*VALUES[\s\S]*`

	t.Run("Успешное добавление пользователя", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		txm := basePSQLInfra.NewTxManager(db)
		repo := psql.NewPSQLUserRepo(txm)

		mock.ExpectExec(insertQueryRegex).
			WithArgs(user.UserID, user.Login, user.PwdHash).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.AddUser(context.Background(), user)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Ошибка: Логин уже занят (Unique Violation 23505)", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		txm := basePSQLInfra.NewTxManager(db)
		repo := psql.NewPSQLUserRepo(txm)

		pgErr := &pgconn.PgError{
			Code: "23505",
		}

		mock.ExpectExec(insertQueryRegex).
			WithArgs(gomockAnyArg(), gomockAnyArg(), gomockAnyArg()).
			WillReturnError(pgErr)

		err = repo.AddUser(context.Background(), user)
		assert.ErrorIs(t, err, usecases.ErrLoginBusy)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Непредвиденная ошибка СУБД при записи", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		txm := basePSQLInfra.NewTxManager(db)
		repo := psql.NewPSQLUserRepo(txm)

		mock.ExpectExec(insertQueryRegex).
			WithArgs(gomockAnyArg(), gomockAnyArg(), gomockAnyArg()).
			WillReturnError(errors.New("disk full"))

		err = repo.AddUser(context.Background(), user)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка сохранения пользователя в БД")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func gomockAnyArg() sqlmock.Argument {
	return sqlmock.AnyArg()
}
