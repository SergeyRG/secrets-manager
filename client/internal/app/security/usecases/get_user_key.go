package usecases

import (
	"context"
	"errors"
	"fmt"

	"github.com/SergeyRG/secrets-manager/client/internal/app/security/domain"
	sharedUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/shared/usecases"
)

type GetUserKeyUseCase struct {
	RemoteKeyRepo domain.EncryptedKeyRepository
	LocalKeyRepo  domain.EncryptedKeyRepository
	st            sharedUsecases.SessionType
}

func NewGetUserKeyUseCase(
	RemoteKeyRepo domain.EncryptedKeyRepository,
	LocalKeyRepo domain.EncryptedKeyRepository,
	st sharedUsecases.SessionType,
) *GetUserKeyUseCase {
	return &GetUserKeyUseCase{
		RemoteKeyRepo: RemoteKeyRepo,
		LocalKeyRepo:  LocalKeyRepo,
		st:            st,
	}
}

func (uc *GetUserKeyUseCase) Execute(ctx context.Context, storage *domain.KeyStorage, pp PasswdProvider) error {
	localEk, localErr := uc.LocalKeyRepo.GetEncryptedKey(ctx)
	if localErr != nil && !errors.Is(localErr, domain.ErrKeyNotFound) {
		return fmt.Errorf("ошибка чтения локального кэша: %w", localErr)
	}

	isCacheEmpty := errors.Is(localErr, domain.ErrKeyNotFound)

	if uc.st == sharedUsecases.SessionTypeLocal {
		if isCacheEmpty {
			return domain.ErrKeyNotFound
		}
		return uc.decryptAndLoad(ctx, storage, pp, localEk)
	}

	remoteEk, remoteErr := uc.RemoteKeyRepo.GetEncryptedKey(ctx)
	if remoteErr != nil {
		if errors.Is(remoteErr, domain.ErrKeyEmpty) {
			return remoteErr
		}

		if isCacheEmpty {
			return fmt.Errorf("сервер недоступен и локальный кэш пуст: %w", remoteErr)
		}

		return uc.decryptAndLoad(ctx, storage, pp, localEk)
	}

	uc.LocalKeyRepo.SaveEncryptedKey(ctx, remoteEk)

	return uc.decryptAndLoad(ctx, storage, pp, remoteEk)
}

func (uc *GetUserKeyUseCase) decryptAndLoad(ctx context.Context, storage *domain.KeyStorage, pp PasswdProvider, ek []byte) error {
	passwd, err := pp.GetPassword(ctx)
	if err != nil {
		return fmt.Errorf("ошибка ввода пароля: %w", err)
	}

	if err := storage.LoadFromEncrypted(ek, passwd); err != nil {
		return fmt.Errorf("не удалось расшифровать хранилище: %w", err)
	}

	return nil
}
