package usecases

import (
	"context"
	"iter"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	sharedUseCases "github.com/SergeyRG/secrets-manager/server/internal/shared/use-cases"
)

type GetUserSecretMetadataPageUseCase struct {
	repo SecretMetadataRepository
	txm  sharedUseCases.TransactionManager
}

func NewGetUserSecretMetadataPageUseCase(
	repo SecretMetadataRepository,
	txm sharedUseCases.TransactionManager,
) *GetUserSecretMetadataPageUseCase {
	return &GetUserSecretMetadataPageUseCase{repo: repo, txm: txm}
}

func (uc *GetUserSecretMetadataPageUseCase) Execute(
	ctx context.Context,
	uID domain.UserID,
	page int,
	perPage int,
) iter.Seq2[secretsDomain.SecretsMetadata, error] {

	if page < 1 {
		page = 1
	}

	if perPage > 20 {
		perPage = 20
	}

	return uc.repo.GetUserSecretsMetadataPage(ctx, uID, page, perPage)
}
