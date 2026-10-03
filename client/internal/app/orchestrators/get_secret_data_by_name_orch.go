package orchestrator

import (
	"context"
	"io"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	secretsUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/usecases"
	securityDomain "github.com/SergeyRG/secrets-manager/client/internal/app/security/domain"
)

type GetSecretDataByNameOrch struct {
	getMetadataUC   *secretsUsecases.GetSecretMetadataUseCase
	getSecretDataUC *secretsUsecases.GetSecretDataUseCase
	ks              *securityDomain.KeyStorage
}

func NewGetSecretDataByNameOrch(
	getMetadataUC *secretsUsecases.GetSecretMetadataUseCase,
	getSecretDataUC *secretsUsecases.GetSecretDataUseCase,
	ks *securityDomain.KeyStorage,
) *GetSecretDataByNameOrch {
	return &GetSecretDataByNameOrch{
		getMetadataUC:   getMetadataUC,
		getSecretDataUC: getSecretDataUC,
		ks:              ks,
	}
}

func (orch *GetSecretDataByNameOrch) Execute(
	ctx context.Context,
	sName string,
	version int,
) (secretsDomain.SecretsMetadata, io.ReadCloser, error) {
	smt, err := orch.getMetadataUC.Execute(ctx, sName, version)
	if err != nil {
		return secretsDomain.SecretsMetadata{}, nil, err
	}

	data, err := orch.getSecretDataUC.Execute(ctx, smt)
	if err != nil {
		return secretsDomain.SecretsMetadata{}, nil, err
	}

	decryptedStream, err := orch.ks.DecryptData(data)
	if err != nil {
		return secretsDomain.SecretsMetadata{}, nil, err
	}

	return smt, decryptedStream, nil
}
