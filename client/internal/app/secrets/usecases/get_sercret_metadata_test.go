package usecases_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	"github.com/SergeyRG/secrets-manager/client/internal/app/secrets/usecases"
	mocks "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/usecases/mocks"
	sharedUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/shared/usecases"

	"go.uber.org/mock/gomock"
)

func TestGetSecretMetadataUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	testSecretName := "api_token"
	testVersion := 1

	expectedMeta := secretsDomain.SecretsMetadata{
		SecretName: testSecretName,
		SecretType: secretsDomain.SecretTypeFreeText,
		Version:    1,
	}

	tests := []struct {
		name         string
		sessionType  sharedUsecases.SessionType
		inputVersion int
		mockRepo     func(local *mocks.MockSecretMetadataRepository, remote *mocks.MockSecretMetadataRepository)
		wantMeta     secretsDomain.SecretsMetadata
		wantErr      error
	}{
		{
			name:         "Ошибка: передан отрицательный номер версии",
			sessionType:  sharedUsecases.SessionTypeRemote,
			inputVersion: -1,
			mockRepo: func(local *mocks.MockSecretMetadataRepository, remote *mocks.MockSecretMetadataRepository) {
				// Валидация происходит до обращения к репозиториям
			},
			wantMeta: secretsDomain.SecretsMetadata{},
			wantErr:  usecases.ErrIncorectSecretVersion,
		},
		{
			name:         "Успешно: метаданные найдены в локальном кэше сразу",
			sessionType:  sharedUsecases.SessionTypeRemote,
			inputVersion: testVersion,
			mockRepo: func(local *mocks.MockSecretMetadataRepository, remote *mocks.MockSecretMetadataRepository) {
				local.EXPECT().
					GetUserSecretMetadataByName(ctx, testSecretName, testVersion).
					Return(expectedMeta, nil).
					Times(1)
			},
			wantMeta: expectedMeta,
			wantErr:  nil,
		},
		{
			name:         "Ошибка: режим Local, метаданных в кэше нет",
			sessionType:  sharedUsecases.SessionTypeLocal,
			inputVersion: testVersion,
			mockRepo: func(local *mocks.MockSecretMetadataRepository, remote *mocks.MockSecretMetadataRepository) {
				local.EXPECT().
					GetUserSecretMetadataByName(ctx, testSecretName, testVersion).
					Return(secretsDomain.SecretsMetadata{}, errors.New("leveldb: not found")).
					Times(1)
			},
			wantMeta: secretsDomain.SecretsMetadata{},
			wantErr:  errors.New("метаданные отсутствуют в локальном кэше"),
		},
		{
			name:         "Успешно: скачивание метаданных с сервера и кэширование",
			sessionType:  sharedUsecases.SessionTypeRemote,
			inputVersion: testVersion,
			mockRepo: func(local *mocks.MockSecretMetadataRepository, remote *mocks.MockSecretMetadataRepository) {
				// 1. В кэше пусто
				local.EXPECT().
					GetUserSecretMetadataByName(ctx, testSecretName, testVersion).
					Return(secretsDomain.SecretsMetadata{}, errors.New("cache miss")).
					Times(1)

				// 2. Идем на сервер
				remote.EXPECT().
					GetUserSecretMetadataByName(ctx, testSecretName, testVersion).
					Return(expectedMeta, nil).
					Times(1)

				// 3. Сохраняем в кэш полученный ответ
				local.EXPECT().
					AddSecretMetadata(ctx, expectedMeta).
					Return(nil).
					Times(1)
			},
			wantMeta: expectedMeta,
			wantErr:  nil,
		},
		{
			name:         "Ошибка: сервера нет, в кэше тоже пусто",
			sessionType:  sharedUsecases.SessionTypeRemote,
			inputVersion: testVersion,
			mockRepo: func(local *mocks.MockSecretMetadataRepository, remote *mocks.MockSecretMetadataRepository) {
				local.EXPECT().
					GetUserSecretMetadataByName(ctx, testSecretName, testVersion).
					Return(secretsDomain.SecretsMetadata{}, errors.New("cache miss")).
					Times(1)

				remote.EXPECT().
					GetUserSecretMetadataByName(ctx, testSecretName, testVersion).
					Return(secretsDomain.SecretsMetadata{}, usecases.ErrServerUnavailable).
					Times(1)
			},
			wantMeta: secretsDomain.SecretsMetadata{},
			wantErr:  errors.New("сервер недоступен, а локальная копия метаданных отсутствует"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLocal := mocks.NewMockSecretMetadataRepository(ctrl)
			mockRemote := mocks.NewMockSecretMetadataRepository(ctrl)

			tt.mockRepo(mockLocal, mockRemote)

			uc := usecases.NewGetSecretMetadataUseCase(mockLocal, mockRemote, tt.sessionType)

			meta, err := uc.Execute(ctx, testSecretName, tt.inputVersion)

			// Проверка ожидаемых ошибок (с учетом %w обертываний)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("ожидалась ошибка %v, но получен nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) && !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("получена ошибка: %q, ожидалась: %q", err.Error(), tt.wantErr.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("не ожидалось ошибки, но получена: %v", err)
			}

			// Проверка структуры возвращаемых метаданных
			if !reflect.DeepEqual(meta, tt.wantMeta) {
				t.Errorf("получены неверные метаданные: %+v, ожидалось: %+v", meta, tt.wantMeta)
			}
		})
	}
}
