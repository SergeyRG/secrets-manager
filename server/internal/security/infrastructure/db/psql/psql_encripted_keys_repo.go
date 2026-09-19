package psql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	securityDomain "github.com/SergeyRG/secrets-manager/server/internal/security/domain"
	securityUseCases "github.com/SergeyRG/secrets-manager/server/internal/security/use-cases"

	sharedUseCases "github.com/SergeyRG/secrets-manager/server/internal/shared/use-cases"
)

type PSQLEncryptedKeyRepo struct {
	txm sharedUseCases.TransactionManager
}

func NewPSQLEncryptedKeyRepo(txm sharedUseCases.TransactionManager) *PSQLEncryptedKeyRepo {
	return &PSQLEncryptedKeyRepo{txm: txm}
}

func (r *PSQLEncryptedKeyRepo) GetUserEncryptedKey(
	ctx context.Context,
	uID domain.UserID,
) (securityDomain.EncryptedKey, error) {
	qe := r.txm.GetExecutor(ctx)
	query := `
			SELECT
			 encrypted_key
			FROM
			 encrypted_keys
			WHERE
			 user_id = $1
			`
	result := qe.QueryRowContext(ctx, query, uID)
	var ek securityDomain.EncryptedKey

	err := result.Scan(&ek)
	if err != nil {
		if errors.Is(sql.ErrNoRows, err) {
			return "", securityUseCases.ErrEncryptionKeyDoesntExists
		}
		return "", err
	}
	return ek, nil
}

func (r *PSQLEncryptedKeyRepo) SetUserEncryptedKey(
	ctx context.Context,
	uID domain.UserID,
	ek securityDomain.EncryptedKey,
) error {
	qe := r.txm.GetExecutor(ctx)
	query := `
			INSERT INTO 
				encrypted_keys (user_id, encrypted_key)
			VALUES
				($1, $2)
			ON CONFLICT
				(user_id) 
			DO UPDATE SET encrypted_key = EXCLUDED.encrypted_key;
			`
	_, err := qe.ExecContext(ctx, query, uID, ek)
	if err != nil {
		return err
	}
	return nil
}
