package usecases

import (
	"context"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	sharedUseCases "github.com/SergeyRG/secrets-manager/server/internal/shared/use-cases"
)

type GetUserSecretMetadataUseCase struct {
	repo SecretMetadataRepository
	txm  sharedUseCases.TransactionManager
}

func NewGetUserSecretMetadataUseCase(
	repo SecretMetadataRepository,
) *GetUserSecretMetadataUseCase {
	return &GetUserSecretMetadataUseCase{repo: repo}
}

func (uc *GetUserSecretMetadataUseCase) Execute(
	ctx context.Context,
	uID domain.UserID,
	secretName string,
	version int,
) (secretsDomain.SecretsMetadata, error) {
	if version < 0 {
		version = 0
	}
	return uc.repo.GetUserSecretMetadataByName(ctx, uID, secretName, version)
}
