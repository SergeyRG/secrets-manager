package fs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	secretsUseCases "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/usecases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFSCacheDataRepository(t *testing.T) {
	// Подготовка общих тестовых данных
	testMetadata := secretsDomain.SecretsMetadata{
		SecretName: "my_super_secret_key",
		Version:    2,
	}
	testContent := "important-secret-payload-data"

	// Вспомогательная функция для ручного расчета имени файла (чтобы проверить корректность buildFileName)
	expectedFileName := func(name string, ver int) string {
		rawKey := name + ":" + strconv.Itoa(ver)
		hash := sha256.Sum256([]byte(rawKey))
		return hex.EncodeToString(hash[:])
	}(testMetadata.SecretName, int(testMetadata.Version))

	t.Run("Успешное сохранение и чтение данных", func(t *testing.T) {
		// Создаем временную изолированную директорию
		tmpDir := t.TempDir()
		repo := NewFSCacheDataRepository(tmpDir)

		// 1. Тестируем AddSecretData
		// Превращаем строку в io.ReadCloser через NopCloser
		inputReader := io.NopCloser(strings.NewReader(testContent))
		err := repo.AddSecretData(context.Background(), testMetadata, inputReader)
		require.NoError(t, err)

		// Дополнительно проверяем, что файл физически создался на диске с правильным хэш-именем
		expectedFilePath := filepath.Join(tmpDir, expectedFileName)
		assert.FileExists(t, expectedFilePath)

		// 2. Тестируем GetSecretData
		outputReader, err := repo.GetSecretData(context.Background(), testMetadata)
		require.NoError(t, err)
		require.NotNil(t, outputReader)
		defer outputReader.Close()

		// Вычитываем данные из возвращенного ReadCloser и проверяем контент
		bytes, err := io.ReadAll(outputReader)
		require.NoError(t, err)
		assert.Equal(t, testContent, string(bytes))
	})

	t.Run("Ошибка: Секрет не найден в кэше", func(t *testing.T) {
		tmpDir := t.TempDir()
		repo := NewFSCacheDataRepository(tmpDir)

		// Пытаемся получить секрет, который мы еще не добавляли
		outputReader, err := repo.GetSecretData(context.Background(), testMetadata)

		// Проверяем, что вернулся nil и специфичная ошибка ErrNotFound
		assert.Nil(t, outputReader)
		assert.ErrorIs(t, err, secretsUseCases.ErrNotFound)
	})

	t.Run("Перезапись существующего секретного файла", func(t *testing.T) {
		tmpDir := t.TempDir()
		repo := NewFSCacheDataRepository(tmpDir)

		// Записываем старые данные
		oldReader := io.NopCloser(strings.NewReader("old_data"))
		err := repo.AddSecretData(context.Background(), testMetadata, oldReader)
		require.NoError(t, err)

		// Записываем новые данные поверх старых (проверка флага os.O_TRUNC)
		newReader := io.NopCloser(strings.NewReader("new_fresh_data"))
		err = repo.AddSecretData(context.Background(), testMetadata, newReader)
		require.NoError(t, err)

		// Проверяем, что в файле остался только новый контент
		outputReader, err := repo.GetSecretData(context.Background(), testMetadata)
		require.NoError(t, err)
		defer outputReader.Close()

		bytes, err := io.ReadAll(outputReader)
		require.NoError(t, err)
		assert.Equal(t, "new_fresh_data", string(bytes))
	})

	t.Run("Ошибка доступа к папке (невалидный путь кэша)", func(t *testing.T) {
		// Передаем заведомо не существующий путь или путь к файлу вместо папки
		repo := NewFSCacheDataRepository("/non-existent-root-dir/cache")

		inputReader := io.NopCloser(strings.NewReader(testContent))
		err := repo.AddSecretData(context.Background(), testMetadata, inputReader)

		// Метод должен вернуть обернутую ошибку создания файла
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка открытия/создания файла")
	})
}
