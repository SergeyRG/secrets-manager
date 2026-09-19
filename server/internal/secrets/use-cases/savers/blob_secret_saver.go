package savers

import (
	"context"
	"io"

	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	usecases "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases"
)

type BlobSecretSaver struct {
	repo usecases.SecretBlobRepository
}

func NewBlobSecretSaver(repo usecases.SecretBlobRepository) *BlobSecretSaver {
	return &BlobSecretSaver{repo: repo}
}

func (s *BlobSecretSaver) SaveSecretData(ctx context.Context, smd secretsDomain.SecretsMetadata, data io.ReadCloser) error {
	return s.repo.AddSecretData(ctx, smd, data)
}
