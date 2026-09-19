package usecases

import (
	"context"
	"errors"

	"github.com/SergeyRG/secrets-manager/client/internal/app/security/domain"
)

type GetUserKeyUseCase struct {
	keyRepo domain.EncryptedKeyRepository
}

func NewGetUserKeyUseCase(
	keyRepo domain.EncryptedKeyRepository,
) *GetUserKeyUseCase {
	return &GetUserKeyUseCase{
		keyRepo: keyRepo,
	}
}

func (uc *GetUserKeyUseCase) Execute(ctx context.Context, storage *domain.KeyStorage, pp PasswdProvider) error {
	ek, err := uc.keyRepo.GetEncryptedKey(ctx)
	if err != nil {
		return err
	}
	passwd, err := pp.GetPassword(ctx)
	if err != nil {
		return errors.New("ошибка ввода пароля")
	}

	return storage.LoadFromEncrypted(ek, passwd)
}
