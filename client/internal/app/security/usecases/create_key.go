package usecases

import (
	"context"
	"errors"

	"github.com/SergeyRG/secrets-manager/client/internal/app/security/domain"
)

var ErrKeyNotExist = errors.New("ключ пользователя не создан")

type CreateKeyUseCase struct {
	keyRepo domain.EncryptedKeyRepository
}

func NewCreateKeyUseCase(
	keyRepo domain.EncryptedKeyRepository,
) *CreateKeyUseCase {
	return &CreateKeyUseCase{
		keyRepo: keyRepo,
	}
}

func (uc *CreateKeyUseCase) Execute(ctx context.Context, pp PasswdProvider, storage *domain.KeyStorage) error {
	err := storage.GenerateNewKey()
	if err != nil {
		return err
	}

	password, err := pp.CreatePassword(ctx)
	if err != nil {
		return err
	}

	ek, err := storage.GetEncryptedKey(password)
	if err != nil {
		return err
	}
	return uc.keyRepo.SaveEncryptedKey(ctx, ek)
}
