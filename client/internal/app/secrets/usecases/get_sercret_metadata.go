package usecases

import (
	"context"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
)

type GetSecretMetadataUseCase struct {
	repo SecretMetadataRepository
}

func NewGetSecretMetadataUseCase(repo SecretMetadataRepository) *GetSecretMetadataUseCase {
	return &GetSecretMetadataUseCase{repo: repo}
}

func (uc *GetSecretMetadataUseCase) Execute(
	ctx context.Context,
	sName string,
	version int,
) (secretsDomain.SecretsMetadata, error) {

	if version < 0 {
		return secretsDomain.SecretsMetadata{}, ErrIncorectSecretVersion
	}

	sm, err := uc.repo.GetUserSecretMetadataByName(ctx, sName, version)
	if err != nil {
		return secretsDomain.SecretsMetadata{}, err
	}
	return sm, nil
}
