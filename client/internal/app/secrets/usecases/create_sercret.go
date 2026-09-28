package usecases

import (
	"context"
	"errors"
	"io"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	sharedUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/shared/usecases"
)

type CreateSecretUseCase struct {
	repo SecretDataRepository
	st   sharedUsecases.SessionType
}

func NewCreateSecretUseCase(
	repo SecretDataRepository,
	st sharedUsecases.SessionType,
) *CreateSecretUseCase {
	return &CreateSecretUseCase{repo: repo, st: st}
}

func (uc *CreateSecretUseCase) Execute(
	ctx context.Context,
	sName string,
	secretType secretsDomain.SecretType,
	data io.ReadCloser,
) error {
	if uc.st == sharedUsecases.SessionTypeLocal {
		return errors.New("нельзя создавать новые секреты в локальной сессии")
	}
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
