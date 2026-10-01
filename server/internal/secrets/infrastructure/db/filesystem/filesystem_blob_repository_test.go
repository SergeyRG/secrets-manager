package filesystem_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	"github.com/SergeyRG/secrets-manager/server/internal/secrets/infrastructure/db/filesystem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretsBlobDataRepo(t *testing.T) {
	testMetadata := secretsDomain.SecretsMetadata{
		SecretID:  domain.SecretID("secret-id-123"),
		VersionID: domain.SecretVersionID("version-uuid-999"),
	}
	testContent := "secret-binary-blob-payload"

	t.Run("Успешное добавление и последующее чтение данных секрета", func(t *testing.T) {
		tmpDir := t.TempDir()
		repo := filesystem.NewSecretsBlobDataRepo(tmpDir)

		src := io.NopCloser(strings.NewReader(testContent))
		err := repo.AddSecretData(context.Background(), testMetadata, src)
		require.NoError(t, err)

		expectedPath := filepath.Join(tmpDir, string(testMetadata.VersionID))
		assert.FileExists(t, expectedPath)

		if runtime.GOOS != "windows" {
			fileInfo, err := os.Stat(expectedPath)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())
		}

		reader, err := repo.GetSecretData(context.Background(), testMetadata)
		require.NoError(t, err)
		require.NotNil(t, reader)
		defer reader.Close()

		bytes, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, testContent, string(bytes))
	})

	t.Run("Ошибка: Попытка перезаписи существующей версии (os.O_EXCL)", func(t *testing.T) {
		tmpDir := t.TempDir()
		repo := filesystem.NewSecretsBlobDataRepo(tmpDir)

		src1 := io.NopCloser(strings.NewReader("initial-data"))
		err := repo.AddSecretData(context.Background(), testMetadata, src1)
		require.NoError(t, err)

		src2 := io.NopCloser(strings.NewReader("overwriting-data"))
		err = repo.AddSecretData(context.Background(), testMetadata, src2)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "не удалось открыть файл")
		assert.True(t, os.IsExist(errorsUnwrapRepo(err))) // Проверяем, что внутренняя ошибка — это os.ErrExist
	})

	t.Run("Ошибка: Чтение несуществующей версии секрета", func(t *testing.T) {
		tmpDir := t.TempDir()
		repo := filesystem.NewSecretsBlobDataRepo(tmpDir)

		missingMetadata := secretsDomain.SecretsMetadata{
			VersionID: domain.SecretVersionID("non-existent-version"),
		}

		reader, err := repo.GetSecretData(context.Background(), missingMetadata)

		assert.Nil(t, reader)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "не удалось открыть файл")
		assert.True(t, os.IsNotExist(errorsUnwrapRepo(err))) // Проверяем, что внутренняя ошибка — это os.ErrNotExist
	})

	t.Run("Ошибка: Запись в несуществующую/заблокированную папку", func(t *testing.T) {
		repo := filesystem.NewSecretsBlobDataRepo("/invalid-system-path-for-test/blob_storage")

		src := io.NopCloser(strings.NewReader("data"))
		err := repo.AddSecretData(context.Background(), testMetadata, src)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "не удалось открыть файл")
	})
}

func errorsUnwrapRepo(err error) error {
	if u, ok := err.(interface{ Unwrap() error }); ok {
		return u.Unwrap()
	}
	return err
}
