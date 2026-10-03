package usecases

import (
	"context"
	"errors"
	"fmt"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	sharedUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/shared/usecases"
)

type GetSecretMetadataUseCase struct {
	LocalRepo  SecretMetadataRepository
	RemoteRepo SecretMetadataRepository
	st         sharedUsecases.SessionType
}

func NewGetSecretMetadataUseCase(
	localRepo SecretMetadataRepository,
	remoteRepo SecretMetadataRepository,
	st sharedUsecases.SessionType,
) *GetSecretMetadataUseCase {
	return &GetSecretMetadataUseCase{
		LocalRepo:  localRepo,
		RemoteRepo: remoteRepo,
		st:         st,
	}
}

func (uc *GetSecretMetadataUseCase) Execute(
	ctx context.Context,
	sName string,
	version int,
) (secretsDomain.SecretsMetadata, error) {

	if version < 0 {
		return secretsDomain.SecretsMetadata{}, ErrIncorectSecretVersion
	}

	sm, err := uc.LocalRepo.GetUserSecretMetadataByName(ctx, sName, version)
	if err == nil {
		return sm, nil
	}

	if uc.st == sharedUsecases.SessionTypeLocal {
		return secretsDomain.SecretsMetadata{}, fmt.Errorf("метаданные отсутствуют в локальном кэше: %w", err)
	}

	remoteSm, err := uc.RemoteRepo.GetUserSecretMetadataByName(ctx, sName, version)
	if err != nil {
		if errors.Is(err, ErrServerUnavailable) {
			return secretsDomain.SecretsMetadata{}, fmt.Errorf("сервер недоступен, а локальная копия метаданных отсутствует: %w", err)
		}
		return secretsDomain.SecretsMetadata{}, err
	}

	_ = uc.LocalRepo.AddSecretMetadata(ctx, remoteSm)

	return remoteSm, nil
}
