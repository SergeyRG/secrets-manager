package filesystem_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/SergeyRG/secrets-manager/server/internal/files/domain"
	"github.com/SergeyRG/secrets-manager/server/internal/files/infrastructure/db/filesystem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilesystemFilesRepo(t *testing.T) {
	fileID := domain.FileID("test-file-id-123")
	fileContent := "hello-world-secret-binary-data"

	t.Run("Успешное сохранение и последующее чтение файла", func(t *testing.T) {
		tmpDir := t.TempDir()
		repo := filesystem.NewFilesystemFilesRepo(tmpDir)

		src := io.NopCloser(strings.NewReader(fileContent))
		err := repo.AddFile(context.Background(), fileID, src)
		require.NoError(t, err)

		expectedPath := filepath.Join(tmpDir, string(fileID))
		assert.FileExists(t, expectedPath)

		if runtime.GOOS != "windows" {
			fileInfo, err := os.Stat(expectedPath)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())
		}

		reader, err := repo.GetFile(context.Background(), fileID)
		require.NoError(t, err)
		require.NotNil(t, reader)
		defer reader.Close()

		bytes, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, fileContent, string(bytes))
	})

	t.Run("Ошибка: Файл с таким ID уже существует (проверка os.O_EXCL)", func(t *testing.T) {
		tmpDir := t.TempDir()
		repo := filesystem.NewFilesystemFilesRepo(tmpDir)

		src1 := io.NopCloser(strings.NewReader("first-data"))
		err := repo.AddFile(context.Background(), fileID, src1)
		require.NoError(t, err)

		src2 := io.NopCloser(strings.NewReader("second-data"))
		err = repo.AddFile(context.Background(), fileID, src2)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "не удалось открыть файл")
		assert.True(t, os.IsExist(errorsUnwrap(err)))
	})

	t.Run("Ошибка: Чтение несуществующего файла", func(t *testing.T) {
		tmpDir := t.TempDir()
		repo := filesystem.NewFilesystemFilesRepo(tmpDir)

		reader, err := repo.GetFile(context.Background(), domain.FileID("missing-id"))

		assert.Nil(t, reader)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "не удалось открыть файл")
		assert.True(t, os.IsNotExist(errorsUnwrap(err)))
	})

	t.Run("Ошибка: Запись в недоступную директорию", func(t *testing.T) {
		repo := filesystem.NewFilesystemFilesRepo("/non-existent-root-path/storage")

		src := io.NopCloser(strings.NewReader("data"))
		err := repo.AddFile(context.Background(), fileID, src)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "не удалось открыть файл")
	})
}

func errorsUnwrap(err error) error {
	if u, ok := err.(interface{ Unwrap() error }); ok {
		return u.Unwrap()
	}
	return err
}
