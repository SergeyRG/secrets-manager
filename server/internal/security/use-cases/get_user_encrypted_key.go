package usecases

import (
	"context"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	securityDomain "github.com/SergeyRG/secrets-manager/server/internal/security/domain"
)

type GetUserEncryptedKeyUseCase struct {
	KeyRepo EncryptedKeyRepo
}

func (uc *GetUserEncryptedKeyUseCase) Execute(ctx context.Context, userID domain.UserID, key securityDomain.EncryptedKey) error {
	return uc.KeyRepo.SetUserEncryptedKey(ctx, userID, key)
}
