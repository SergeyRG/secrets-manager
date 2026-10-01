package fs_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergeyRG/secrets-manager/client/internal/app/security/domain"
	"github.com/SergeyRG/secrets-manager/client/internal/app/security/infra/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	// Убедитесь, что путь импорта совпадает с вашим go.mod
)

func TestFSEncryptedKeyRepo(t *testing.T) {
	testData := []byte("my-super-secret-encrypted-key-bytes")

	t.Run("Успешное сохранение и последующее чтение ключа", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "encrypted_key.dat")

		repo := fs.NewFSEncryptedKeyRepo(filePath)

		err := repo.SaveEncryptedKey(context.Background(), testData)
		require.NoError(t, err)

		assert.FileExists(t, filePath)

		savedKey, err := repo.GetEncryptedKey(context.Background())
		require.NoError(t, err)
		assert.Equal(t, testData, savedKey)
	})

	t.Run("Ошибка: Файл ключа не существует", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonExistentFile := filepath.Join(tmpDir, "missing_key.dat")

		repo := fs.NewFSEncryptedKeyRepo(nonExistentFile)

		key, err := repo.GetEncryptedKey(context.Background())

		assert.Nil(t, key)
		assert.EqualError(t, err, "ошибка чтения ключа из локального кэша")
	})

	t.Run("Ошибка: Файл пустой (длина ключа 0)", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "empty_key.dat")

		err := os.WriteFile(filePath, []byte(""), 0600)
		require.NoError(t, err)

		repo := fs.NewFSEncryptedKeyRepo(filePath)

		key, err := repo.GetEncryptedKey(context.Background())

		assert.Nil(t, key)
		assert.ErrorIs(t, err, domain.ErrKeyNotFound)
	})

	t.Run("Ошибка записи в файл (невалидная директория)", func(t *testing.T) {
		repo := fs.NewFSEncryptedKeyRepo("/non-existent-folder/key.dat")

		err := repo.SaveEncryptedKey(context.Background(), testData)

		assert.EqualError(t, err, "ошибка сохранения ключа в локальном кэше")
	})
}
