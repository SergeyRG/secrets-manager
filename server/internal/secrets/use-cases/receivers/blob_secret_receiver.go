package savers

import (
	"context"
	"io"

	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	usecases "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases"
)

type BlobSecretReceiver struct {
	repo usecases.SecretBlobRepository
}

func NewBlobSecretReceiver(repo usecases.SecretBlobRepository) *BlobSecretReceiver {
	return &BlobSecretReceiver{repo: repo}
}

func (s *BlobSecretReceiver) ReceiveSecretData(ctx context.Context, smd secretsDomain.SecretsMetadata) (io.ReadCloser, error) {
	return s.repo.GetSecretData(ctx, smd)
}
