package domain_test

import (
	"testing"
	"time"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretType_ToString_And_FromString(t *testing.T) {
	t.Run("Кросс-проверка всех типов через ToString и FromString", func(t *testing.T) {
		types := []secretsDomain.SecretType{
			secretsDomain.SecretTypeBinary,
			secretsDomain.SecretTypeFreeText,
			secretsDomain.SecretTypeAuthData,
			secretsDomain.SecretTypeBankCard,
			secretsDomain.SecretTypeUnknown,
		}

		for _, st := range types {
			str := st.ToString()
			parsedType := secretsDomain.SecretTypeFromString(str)
			assert.Equal(t, st, parsedType, "Несовпадение при конвертации типа: %s", str)
		}
	})

	t.Run("FromString преобразует строки в любом регистре", func(t *testing.T) {
		assert.Equal(t, secretsDomain.SecretTypeBinary, secretsDomain.SecretTypeFromString("secret_type_binary"))
		assert.Equal(t, secretsDomain.SecretTypeBankCard, secretsDomain.SecretTypeFromString("SeCrEt_TyPe_BaNk_CaRd"))
	})

	t.Run("Невалидная строка или дефолтное состояние возвращает SecretTypeUnknown", func(t *testing.T) {
		assert.Equal(t, secretsDomain.SecretTypeUnknown, secretsDomain.SecretTypeFromString("INVALID_STRING_TYPE"))

		var invalidType secretsDomain.SecretType = 999
		assert.Equal(t, "SECRET_TYPE_UNKNOWN", invalidType.ToString())
	})
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name       string
		secretName string
		wantErr    bool
	}{
		{"Валидное имя: только буквы", "mySecretName", false},
		{"Валидное имя: с цифрами", "secret123", false},
		{"Валидное имя: с нижним подчеркиванием", "my_secret_key", false},
		{"Валидное имя: смешанное", "Secret_123_Key", false},
		{"Невалидное имя: содержит дефис", "my-secret-key", true},
		{"Невалидное имя: содержит пробел", "my secret key", true},
		{"Невалидное имя: спецсимволы", "secret!", true},
		{"Невалидное имя: русские буквы", "секрет", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := secretsDomain.ValidateName(tt.secretName)
			if tt.wantErr {
				assert.ErrorIs(t, err, secretsDomain.ErrInvalidNameFormat)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewSecretMetadata(t *testing.T) {
	uID := domain.UserID("test-user-id")

	t.Run("Успешная генерация метаданных для новой записи", func(t *testing.T) {
		name := "valid_binary_secret_name"
		metadata, err := secretsDomain.NewSecretMetadata(uID, secretsDomain.SecretTypeBinary, name)

		require.NoError(t, err)
		assert.Equal(t, uID, metadata.UserID)
		assert.Equal(t, name, metadata.SecretName)
		assert.Equal(t, secretsDomain.SecretTypeBinary, metadata.SecretType)
		assert.Equal(t, int64(1), metadata.Version)

		assert.NotEmpty(t, metadata.SecretID)
		assert.NotEmpty(t, metadata.VersionID)
		assert.WithinDuration(t, time.Now().UTC(), metadata.TimeCreation, 2*time.Second)
	})

	t.Run("Ошибка: Пустое имя секрета", func(t *testing.T) {
		_, err := secretsDomain.NewSecretMetadata(uID, secretsDomain.SecretTypeBinary, "")
		assert.ErrorIs(t, err, secretsDomain.ErrEmptySecretName)
	})

	t.Run("Ошибка: Нарушение регулярного выражения формата имени", func(t *testing.T) {
		_, err := secretsDomain.NewSecretMetadata(uID, secretsDomain.SecretTypeBinary, "archive.tar.gz")
		assert.ErrorIs(t, err, secretsDomain.ErrInvalidNameFormat)
	})

	t.Run("Ошибка: Невалидный или граничный тип секрета", func(t *testing.T) {
		// ИСПРАВЛЕНО: Заменили зарезервированное поле 'type' на 'secretType'
		tests := []struct {
			name       string
			secretType secretsDomain.SecretType
		}{
			{name: "Тип Unknown (минимальная граница)", secretType: secretsDomain.SecretTypeUnknown},
			{name: "Тип Max (максимальная граница)", secretType: secretsDomain.SecretTypeMax},
			{name: "Тип сильно выше максимального", secretType: secretsDomain.SecretTypeMax + 10},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := secretsDomain.NewSecretMetadata(uID, tt.secretType, "correct_name")
				assert.ErrorIs(t, err, secretsDomain.ErrInvalidType)
			})
		}
	})
}

func TestNewSecretVersionMetadata(t *testing.T) {
	t.Run("Успешное создание метаданных новой версии (инкремент)", func(t *testing.T) {
		uID := domain.UserID("user-id-xyz")
		initialMetadata, err := secretsDomain.NewSecretMetadata(uID, secretsDomain.SecretTypeFreeText, "my_notes")
		require.NoError(t, err)

		time.Sleep(5 * time.Millisecond)

		updatedMetadata, err := secretsDomain.NewSecretVersionMetadata(initialMetadata)
		require.NoError(t, err)

		assert.Equal(t, initialMetadata.SecretID, updatedMetadata.SecretID)
		assert.Equal(t, initialMetadata.SecretName, updatedMetadata.SecretName)
		assert.Equal(t, initialMetadata.UserID, updatedMetadata.UserID)
		assert.Equal(t, initialMetadata.SecretType, updatedMetadata.SecretType)

		assert.NotEmpty(t, updatedMetadata.VersionID)
		assert.NotEqual(t, initialMetadata.VersionID, updatedMetadata.VersionID)

		assert.Equal(t, initialMetadata.Version+1, updatedMetadata.Version)

		assert.True(t, updatedMetadata.TimeCreation.After(initialMetadata.TimeCreation))
		assert.WithinDuration(t, time.Now().UTC(), updatedMetadata.TimeCreation, 2*time.Second)
	})
}
