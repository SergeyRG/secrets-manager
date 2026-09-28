package usecases

import (
	"context"
	"errors"
	"fmt"
	"io"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	sharedUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/shared/usecases"
)

type GetSecretDataUseCase struct {
	RemoteDataRepo    SecretDataRepository
	LocalDataRepo     SecretDataRepository
	LocalMetadataRepo SecretMetadataRepository
	st                sharedUsecases.SessionType
}

func NewGetSecretDataUseCase(
	RemoteDataRepo SecretDataRepository,
	LocalDataRepo SecretDataRepository,
	LocalMetadataRepo SecretMetadataRepository,
	st sharedUsecases.SessionType,
) *GetSecretDataUseCase {
	return &GetSecretDataUseCase{
		RemoteDataRepo:    RemoteDataRepo,
		LocalDataRepo:     LocalDataRepo,
		LocalMetadataRepo: LocalMetadataRepo,
		st:                st,
	}
}

func (uc *GetSecretDataUseCase) Execute(
	ctx context.Context,
	smt secretsDomain.SecretsMetadata,
) (io.ReadCloser, error) {
	localData, err := uc.LocalDataRepo.GetSecretData(ctx, smt)
	if err == nil {
		return localData, nil
	}
	if uc.st == sharedUsecases.SessionTypeLocal {
		return nil, err
	}

	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	remoteData, err := uc.RemoteDataRepo.GetSecretData(ctx, smt)
	if err != nil {
		if errors.Is(err, ErrServerUnavailable) {
			return nil, fmt.Errorf("сервер недоступен, а запрашиваемая версия секрета еще не была закэширована: %w", err)
		}
		return nil, err
	}

	defer remoteData.Close()

	err = uc.LocalDataRepo.AddSecretData(ctx, smt, remoteData)
	if err != nil {
		return nil, fmt.Errorf("не удалось сохранить бинарные данные в кэш: %w", err)
	}

	err = uc.LocalMetadataRepo.AddSecretMetadata(ctx, smt)
	if err != nil {
		return nil, fmt.Errorf("не удалось обновить индекс метаданных кэша: %w", err)
	}

	localData, err = uc.LocalDataRepo.GetSecretData(ctx, smt)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия свежего файла кэша: %w", err)
	}

	return localData, nil
}
