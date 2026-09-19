package usecases

import (
	"context"
	"io"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
)

type GetSecretDataUseCase struct {
	repo SecretDataRepository
}

func NewGetSecretDataUseCase(repo SecretDataRepository) *GetSecretDataUseCase {
	return &GetSecretDataUseCase{repo: repo}
}

func (uc *GetSecretDataUseCase) Execute(
	ctx context.Context,
	smt secretsDomain.SecretsMetadata,
) (io.ReadCloser, error) {

	data, err := uc.repo.GetSecretData(ctx, smt)
	if err != nil {
		return nil, err
	}
	return data, nil
}
