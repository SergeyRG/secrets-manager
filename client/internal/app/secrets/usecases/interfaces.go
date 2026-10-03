//go:generate mockgen -source=$GOFILE -destination=mocks/mocks.go -package=mockspackage usecases

package usecases

import (
	"context"
	"io"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
)

type SecretMetadataRepository interface {
	AddSecretMetadata(context.Context, secretsDomain.SecretsMetadata) error
	GetUserSecretMetadataByName(ctx context.Context, sName string, version int) (secretsDomain.SecretsMetadata, error)
	GetUserSecretsMetadataPage(ctx context.Context, page int, perPage int) ([]secretsDomain.SecretsMetadata, error)
}

type SecretDataRepository interface {
	AddSecretData(ctx context.Context, smt secretsDomain.SecretsMetadata, data io.ReadCloser) error
	GetSecretData(ctx context.Context, smt secretsDomain.SecretsMetadata) (io.ReadCloser, error)
}
