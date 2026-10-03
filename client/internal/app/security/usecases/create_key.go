package usecases

import (
	"context"
	"errors"
	"fmt"

	"github.com/SergeyRG/secrets-manager/client/internal/app/security/domain"
	sharedUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/shared/usecases"
)

var (
	ErrKeyNotExist         = errors.New("ключ пользователя не создан")
	ErrKeyDontSavedInCache = errors.New("ошибка сохранения ключа в локальном кэше")
)

type CreateKeyUseCase struct {
	RemoteKeyRepo domain.EncryptedKeyRepository
	LocalKeyRepo  domain.EncryptedKeyRepository
	st            sharedUsecases.SessionType
}

func NewCreateKeyUseCase(
	RemoteKeyRepo domain.EncryptedKeyRepository,
	LocalKeyRepo domain.EncryptedKeyRepository,
	st sharedUsecases.SessionType,
) *CreateKeyUseCase {
	return &CreateKeyUseCase{
		RemoteKeyRepo: RemoteKeyRepo,
		LocalKeyRepo:  LocalKeyRepo,
		st:            st,
	}
}

func (uc *CreateKeyUseCase) Execute(ctx context.Context, pp PasswdProvider, storage *domain.KeyStorage) error {
	if uc.st == sharedUsecases.SessionTypeLocal {
		return errors.New("создание ключа в локальной сессии недопустимо")
	}
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
	err = uc.RemoteKeyRepo.SaveEncryptedKey(ctx, ek)
	if err != nil {
		return err
	}
	err = uc.LocalKeyRepo.SaveEncryptedKey(ctx, ek)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrKeyDontSavedInCache, err)
	}
	return nil
}
