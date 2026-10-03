package psql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	authDomain "github.com/SergeyRG/secrets-manager/server/internal/auth/domain"
	usecases "github.com/SergeyRG/secrets-manager/server/internal/auth/use-cases"
	basePSQLInfra "github.com/SergeyRG/secrets-manager/server/internal/shared/infrastructure/psql"
	"github.com/jackc/pgx/v5/pgconn"
)

type PSQLUserRepo struct {
	basePSQLInfra.BaseRepository
}

func NewPSQLUserRepo(txm *basePSQLInfra.PSQLTxManager) *PSQLUserRepo {
	return &PSQLUserRepo{
		BaseRepository: basePSQLInfra.BaseRepository{TxManager: txm},
	}
}

func (repo *PSQLUserRepo) GetUserByLogin(ctx context.Context, login string) (*authDomain.User, error) {
	qe := repo.GetExecutor(ctx)
	query := `SELECT user_id, login, pwd_hash
			FROM users
			WHERE login = $1`

	sqlResults := qe.QueryRowContext(ctx, query, login)

	user := authDomain.User{}
	err := sqlResults.Scan(&user.UserID, &user.Login, &user.PwdHash)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, usecases.ErrWrongLoginOrPassword
		}
		return nil, fmt.Errorf(
			"ошибка получения пользователя из БД: %w", err)
	}

	return &user, nil
}

func (repo *PSQLUserRepo) AddUser(ctx context.Context, user authDomain.User) error {
	qe := repo.GetExecutor(ctx)
	query := `
			INSERT INTO users (user_id, login, pwd_hash)
			VALUES ($1, $2, $3);`

	_, err := qe.ExecContext(ctx, query, user.UserID, user.Login, user.PwdHash)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return usecases.ErrLoginBusy
		}
		return fmt.Errorf(
			"ошибка сохранения пользователя в БД: %w", err)
	}

	return nil
}
