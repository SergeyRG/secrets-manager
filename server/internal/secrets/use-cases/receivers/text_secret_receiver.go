package savers

import (
	"bytes"
	"context"
	"fmt"
	"io"

	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	usecases "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases"
)

type TextSecretReceiver struct {
	repo usecases.SecretTextDataRepository
}

func NewTextSecretReceiver(repo usecases.SecretTextDataRepository) *TextSecretReceiver {
	return &TextSecretReceiver{repo: repo}
}

func (s *TextSecretReceiver) ReceiveSecretData(ctx context.Context, smd secretsDomain.SecretsMetadata) (io.ReadCloser, error) {
	data, err := s.repo.GetSecretData(ctx, smd)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения данных секрета из репозитория: %w", err)
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}
