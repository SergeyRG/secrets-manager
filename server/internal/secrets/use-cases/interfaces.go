//go:generate mockgen -destination=mocks/mocks.go -package=mocks github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases SecretMetadataRepository,SecretTextDataRepository,SecretBlobRepository,SecretDataSaver,SecretDataReceiver
package usecases

import (
	"context"
	"io"
	"iter"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
)

type SecretMetadataRepository interface {
	AddSecretMetadata(context.Context, secretsDomain.SecretsMetadata) error
	GetUserSecretMetadataByName(ctx context.Context, uId domain.UserID, sName string, version int) (secretsDomain.SecretsMetadata, error)
	GetUserSecretsMetadataPage(ctx context.Context, uID domain.UserID, page int, perPage int) iter.Seq2[secretsDomain.SecretsMetadata, error]
}

type SecretTextDataRepository interface {
	AddSecretData(ctx context.Context, smt secretsDomain.SecretsMetadata, data []byte) error
	GetSecretData(ctx context.Context, smt secretsDomain.SecretsMetadata) ([]byte, error)
}

type SecretBlobRepository interface {
	AddSecretData(ctx context.Context, smt secretsDomain.SecretsMetadata, data io.ReadCloser) error
	GetSecretData(ctx context.Context, smt secretsDomain.SecretsMetadata) (io.ReadCloser, error)
}

type SecretDataSaver interface {
	SaveSecretData(ctx context.Context, smd secretsDomain.SecretsMetadata, data io.ReadCloser) error
}

type SecretDataReceiver interface {
	ReceiveSecretData(ctx context.Context, smd secretsDomain.SecretsMetadata) (io.ReadCloser, error)
}
