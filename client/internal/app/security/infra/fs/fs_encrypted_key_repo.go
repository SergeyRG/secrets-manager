package fs

import (
	"context"
	"errors"
	"os"

	"github.com/SergeyRG/secrets-manager/client/internal/app/security/domain"
)

type FSEncryptedKeyRepo struct {
	filePath string
}

func NewFSEncryptedKeyRepo(
	filePath string,
) *FSEncryptedKeyRepo {
	return &FSEncryptedKeyRepo{
		filePath: filePath,
	}
}

func (lc *FSEncryptedKeyRepo) SaveEncryptedKey(ctx context.Context, data []byte) error {
	err := os.WriteFile(lc.filePath, data, 0600)
	if err != nil {
		return errors.New("ошибка сохранения ключа в локальном кэше")
	}
	return nil
}

func (lc *FSEncryptedKeyRepo) GetEncryptedKey(ctx context.Context) ([]byte, error) {
	key, err := os.ReadFile(lc.filePath)
	if err != nil {
		return nil, errors.New("ошибка чтения ключа из локального кэша")
	}
	if len(key) == 0 {
		return nil, domain.ErrKeyNotFound
	}

	return key, nil
}
