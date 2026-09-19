package usecases

import (
	"context"
	"io"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
)

type CreateSecretUseCase struct {
	repo SecretDataRepository
}

func NewCreateSecretUseCase(repo SecretDataRepository) *CreateSecretUseCase {
	return &CreateSecretUseCase{repo: repo}
}

func (uc *CreateSecretUseCase) Execute(
	ctx context.Context,
	sName string,
	secretType secretsDomain.SecretType,
	data io.ReadCloser,
) error {
	smd := secretsDomain.SecretsMetadata{
		SecretName: sName,
		SecretType: secretType,
		Version:    0,
	}

	err := uc.repo.AddSecretData(ctx, smd, data)
	if err != nil {
		return err
	}
	return nil
}
