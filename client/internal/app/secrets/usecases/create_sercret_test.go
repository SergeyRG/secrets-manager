package usecases_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	"github.com/SergeyRG/secrets-manager/client/internal/app/secrets/usecases"
	mocks "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/usecases/mocks"
	sharedUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/shared/usecases"

	"go.uber.org/mock/gomock"
)

func TestCreateSecretUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	testSecretName := "my_test_secret"
	testSecretType := secretsDomain.SecretTypeFreeText
	testPayload := "some_raw_secret_data"

	// Ожидаемые метаданные, которые юзкейс должен сформировать внутри себя
	expectedMetadata := secretsDomain.SecretsMetadata{
		SecretName: testSecretName,
		SecretType: testSecretType,
		Version:    0,
	}

	tests := []struct {
		name        string
		sessionType sharedUsecases.SessionType
		mockRepo    func(m *mocks.MockSecretDataRepository, stream io.ReadCloser)
		wantErr     error
	}{
		{
			name:        "Ошибка: попытка создания секрета в локальной сессии",
			sessionType: sharedUsecases.SessionTypeLocal,
			mockRepo: func(m *mocks.MockSecretDataRepository, stream io.ReadCloser) {
				// В локальной сессии репозиторий вызываться не должен
			},
			wantErr: errors.New("нельзя создавать новые секреты в локальной сессии"),
		},
		{
			name:        "Успешный сценарий в удаленной сессии",
			sessionType: sharedUsecases.SessionTypeRemote,
			mockRepo: func(m *mocks.MockSecretDataRepository, stream io.ReadCloser) {
				// Проверяем, что в репозиторий уходят правильные метаданные и тот же поток данных
				m.EXPECT().
					AddSecretData(ctx, expectedMetadata, stream).
					Return(nil).
					Times(1)
			},
			wantErr: nil,
		},
		{
			name:        "Ошибка репозитория пробрасывается наверх",
			sessionType: sharedUsecases.SessionTypeRemote,
			mockRepo: func(m *mocks.MockSecretDataRepository, stream io.ReadCloser) {
				m.EXPECT().
					AddSecretData(ctx, expectedMetadata, stream).
					Return(errors.New("network timeout")).
					Times(1)
			},
			wantErr: errors.New("network timeout"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// 1. Создаем мок репозитория
			mockRepo := mocks.NewMockSecretDataRepository(ctrl)

			// Подготавливаем поток данных для этой итерации теста
			stream := io.NopCloser(bytes.NewBufferString(testPayload))

			// Настраиваем поведение мока
			tt.mockRepo(mockRepo, stream)

			// 2. Инициализируем тестируемый юзкейс
			uc := usecases.NewCreateSecretUseCase(mockRepo, tt.sessionType)

			// 3. Выполняем бизнес-логику
			err := uc.Execute(ctx, testSecretName, testSecretType, stream)

			// 4. Проверяем результаты
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("ожидалась ошибка %v, но получен nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("получена ошибка: %q, ожидалась: %q", err.Error(), tt.wantErr.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("не ожидалось ошибки, но получена: %v", err)
			}
		})
	}
}
