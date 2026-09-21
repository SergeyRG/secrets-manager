package usecases

import (
	"context"
	"io"
	"iter"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
)

type SecretMetadataRepository interface {
	AddSecretMetadata(context.Context, secretsDomain.SecretsMetadata) error
	GetUserSecretMetadataByName(ctx context.Context, sName string, version int) (secretsDomain.SecretsMetadata, error)
	GetUserSecretsMetadataPage(ctx context.Context, page int, perPage int) iter.Seq2[secretsDomain.SecretsMetadata, error]
}

type SecretDataRepository interface {
	AddSecretData(ctx context.Context, smt secretsDomain.SecretsMetadata, data io.ReadCloser) error
	GetSecretData(ctx context.Context, smt secretsDomain.SecretsMetadata) (io.ReadCloser, error)
}
