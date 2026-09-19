package usecases

import (
	"context"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	securityDomain "github.com/SergeyRG/secrets-manager/server/internal/security/domain"
)

type EncryptedKeyRepo interface {
	GetUserEncryptedKey(context.Context, domain.UserID) (securityDomain.EncryptedKey, error)
	SetUserEncryptedKey(context.Context, domain.UserID, securityDomain.EncryptedKey) error
}
