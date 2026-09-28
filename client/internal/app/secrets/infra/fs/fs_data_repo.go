package fs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	secretsUseCases "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/usecases"
)

type FSCacheDataRepository struct {
	path string
}

func NewFSCacheDataRepository(path string) *FSCacheDataRepository {
	return &FSCacheDataRepository{path: path}
}

func (r *FSCacheDataRepository) AddSecretData(
	ctx context.Context,
	smt secretsDomain.SecretsMetadata,
	data io.ReadCloser,
) error {
	defer data.Close()
	fileName := r.buildFileName(smt.SecretName, int(smt.Version))
	fullPath := filepath.Join(r.path, fileName)

	f, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("ошибка открытия/создания файла: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, data)
	if err != nil {
		return fmt.Errorf("ошибка записи данных в файл кэша: %w", err)
	}

	err = f.Close()
	if err != nil {
		return fmt.Errorf("ошибка закрытия файла после записи: %w", err)
	}
	return nil
}

func (r *FSCacheDataRepository) GetSecretData(
	ctx context.Context,
	smt secretsDomain.SecretsMetadata,
) (io.ReadCloser, error) {

	fileName := r.buildFileName(smt.SecretName, int(smt.Version))
	fullPath := filepath.Join(r.path, fileName)

	f, err := os.OpenFile(fullPath, os.O_RDONLY, 0644)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, secretsUseCases.ErrNotFound
		}
		return nil, fmt.Errorf("ошибка доступа к файлу кэша: %w", err)
	}

	return f, nil
}

func (r *FSCacheDataRepository) buildFileName(secretName string, version int) string {
	rawKey := secretName + ":" + strconv.Itoa(version)
	sNameHash := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sNameHash[:])
}
