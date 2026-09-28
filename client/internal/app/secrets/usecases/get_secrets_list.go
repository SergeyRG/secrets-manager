package usecases

import (
	"context"
	"errors"
	"fmt"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	sharedUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/shared/usecases"
)

type GetSecretsListUseCase struct {
	remoteRepo SecretMetadataRepository
	localRepo  SecretMetadataRepository
	st         sharedUsecases.SessionType
}

func NewGetSecretsListUseCase(
	remote SecretMetadataRepository,
	local SecretMetadataRepository,
	st sharedUsecases.SessionType,
) *GetSecretsListUseCase {
	return &GetSecretsListUseCase{remoteRepo: remote, localRepo: local, st: st}
}

func (uc *GetSecretsListUseCase) Execute(
	ctx context.Context,
	page int,
	perPage int,
) ([]secretsDomain.SecretsMetadata, error) {

	if uc.st == sharedUsecases.SessionTypeLocal {
		return uc.readLocalCache(ctx, page, perPage)
	}

	list, err := uc.remoteRepo.GetUserSecretsMetadataPage(ctx, page, perPage)
	if err == nil {
		for _, item := range list {
			_ = uc.localRepo.AddSecretMetadata(ctx, item)
		}
		return list, nil
	}

	if errors.Is(err, ErrServerUnavailable) {
		localList, localErr := uc.readLocalCache(ctx, page, perPage)
		if localErr != nil {
			return nil, localErr
		}
		return localList, ErrDataFromLocalCache
	}

	return nil, err
}

func (uc *GetSecretsListUseCase) readLocalCache(ctx context.Context, page, perPage int) ([]secretsDomain.SecretsMetadata, error) {
	localList, err := uc.localRepo.GetUserSecretsMetadataPage(ctx, page, perPage)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения локального индекса метаданных: %w", err)
	}
	return localList, nil
}
