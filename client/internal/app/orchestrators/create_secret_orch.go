package orchestrator

import (
	"context"
	"io"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	secretsUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/usecases"
	securityDomain "github.com/SergeyRG/secrets-manager/client/internal/app/security/domain"
)

type CreateSecretOrch struct {
	createSecretUC *secretsUsecases.CreateSecretUseCase
	ks             *securityDomain.KeyStorage
}

func NewCreateSecretOrch(
	createSecretUC *secretsUsecases.CreateSecretUseCase,
	ks *securityDomain.KeyStorage,
) *CreateSecretOrch {
	return &CreateSecretOrch{
		createSecretUC: createSecretUC,
		ks:             ks,
	}
}

func (orch *CreateSecretOrch) Execute(
	ctx context.Context,
	sName string,
	secretType secretsDomain.SecretType,
	data io.ReadCloser,
) error {
	encryptedStream, err := orch.ks.EncryptData(data)
	if err != nil {
		return err
	}
	err = orch.createSecretUC.Execute(ctx, sName, secretType, encryptedStream)
	if err != nil {
		return err
	}

	return nil
}
