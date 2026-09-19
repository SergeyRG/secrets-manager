package filesystem

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
)

type SecretsBlobDataRepo struct {
	baseDir string
}

func NewSecretsBlobDataRepo(baseDir string) *SecretsBlobDataRepo {
	return &SecretsBlobDataRepo{
		baseDir: baseDir,
	}
}

func (r *SecretsBlobDataRepo) AddSecretData(ctx context.Context, sm secretsDomain.SecretsMetadata, data io.ReadCloser) error {
	defer data.Close()

	path := filepath.Join(r.baseDir, string(sm.VersionID))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("не удалось открыть файл: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, data)
	if err != nil {
		return fmt.Errorf("ошибка сохранения файла: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("ошибка закрытия файла после записи: %w", err)
	}
	return nil
}
func (r *SecretsBlobDataRepo) GetSecretData(ctx context.Context, sm secretsDomain.SecretsMetadata) (io.ReadCloser, error) {
	path := filepath.Join(r.baseDir, string(sm.VersionID))
	file, err := os.OpenFile(path, os.O_RDONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл: %w", err)
	}

	return file, nil
}
