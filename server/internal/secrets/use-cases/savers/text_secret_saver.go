package savers

import (
	"context"
	"fmt"
	"io"

	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	usecases "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases"
)

type TextSecretSaver struct {
	repo usecases.SecretTextDataRepository
}

func NewTextSecretSaver(repo usecases.SecretTextDataRepository) *TextSecretSaver {
	return &TextSecretSaver{repo: repo}
}

func (s *TextSecretSaver) SaveSecretData(ctx context.Context, smd secretsDomain.SecretsMetadata, data io.ReadCloser) error {
	defer data.Close()
	dataBytes, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("ошибка чтения данных: %w", err)
	}
	return s.repo.AddSecretData(ctx, smd, dataBytes)
}
