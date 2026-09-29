package usecases_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	"github.com/SergeyRG/secrets-manager/client/internal/app/secrets/usecases"
	mocks "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/usecases/mocks"
	sharedUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/shared/usecases"

	"go.uber.org/mock/gomock"
)

func TestGetSecretDataUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	testMeta := secretsDomain.SecretsMetadata{
		SecretName: "my_key",
		SecretType: secretsDomain.SecretTypeFreeText,
		Version:    1,
	}
	secretContent := "very_secure_content"

	tests := []struct {
		name        string
		sessionType sharedUsecases.SessionType
		mockRepo    func(remote *mocks.MockSecretDataRepository, localData *mocks.MockSecretDataRepository, localMeta *mocks.MockSecretMetadataRepository)
		wantData    string
		wantErr     error
	}{
		{
			name:        "Успешно: данные найдены в локальном кэше сразу",
			sessionType: sharedUsecases.SessionTypeRemote,
			mockRepo: func(remote *mocks.MockSecretDataRepository, localData *mocks.MockSecretDataRepository, localMeta *mocks.MockSecretMetadataRepository) {
				stream := io.NopCloser(bytes.NewBufferString(secretContent))
				// Первый вызов GetSecretData успешен — сразу возвращаем поток
				localData.EXPECT().GetSecretData(ctx, testMeta).Return(stream, nil).Times(1)
			},
			wantData: secretContent,
			wantErr:  nil,
		},
		{
			name:        "Ошибка: Локальная сессия, данных нет в кэше",
			sessionType: sharedUsecases.SessionTypeLocal,
			mockRepo: func(remote *mocks.MockSecretDataRepository, localData *mocks.MockSecretDataRepository, localMeta *mocks.MockSecretMetadataRepository) {
				// В оффлайн-режиме при ErrNotFound возвращаем ошибку наружу
				localData.EXPECT().GetSecretData(ctx, testMeta).Return(nil, usecases.ErrNotFound).Times(1)
			},
			wantData: "",
			wantErr:  usecases.ErrNotFound,
		},
		{
			name:        "Успешно: скачивание с сервера, кэширование и открытие нового потока из файла",
			sessionType: sharedUsecases.SessionTypeRemote,
			mockRepo: func(remote *mocks.MockSecretDataRepository, localData *mocks.MockSecretDataRepository, localMeta *mocks.MockSecretMetadataRepository) {
				// 1. Первый вызов кэша: данных нет
				localData.EXPECT().GetSecretData(ctx, testMeta).Return(nil, usecases.ErrNotFound).Times(1)

				// 2. Запрос к серверу: получаем удаленный поток
				serverStream := io.NopCloser(bytes.NewBufferString(secretContent))
				remote.EXPECT().GetSecretData(ctx, testMeta).Return(serverStream, nil).Times(1)

				// 3. Сохранение бинарных данных на диск и обновление индекса
				localData.EXPECT().AddSecretData(ctx, testMeta, gomock.Any()).Return(nil).Times(1)
				localMeta.EXPECT().AddSecretMetadata(ctx, testMeta).Return(nil).Times(1)

				// 4. Эмулируем ваше поведение: после записи открывается новый поток из файла кэша
				freshLocalStream := io.NopCloser(bytes.NewBufferString(secretContent))
				localData.EXPECT().GetSecretData(ctx, testMeta).Return(freshLocalStream, nil).Times(1)
			},
			wantData: secretContent,
			wantErr:  nil,
		},
		{
			name:        "Ошибка: сервера нет, в кэше тоже пусто",
			sessionType: sharedUsecases.SessionTypeRemote,
			mockRepo: func(remote *mocks.MockSecretDataRepository, localData *mocks.MockSecretDataRepository, localMeta *mocks.MockSecretMetadataRepository) {
				localData.EXPECT().GetSecretData(ctx, testMeta).Return(nil, usecases.ErrNotFound).Times(1)
				remote.EXPECT().GetSecretData(ctx, testMeta).Return(nil, usecases.ErrServerUnavailable).Times(1)
			},
			wantData: "",
			wantErr:  usecases.ErrServerUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRemote := mocks.NewMockSecretDataRepository(ctrl)
			mockLocalData := mocks.NewMockSecretDataRepository(ctrl)
			mockLocalMeta := mocks.NewMockSecretMetadataRepository(ctrl)

			tt.mockRepo(mockRemote, mockLocalData, mockLocalMeta)

			uc := usecases.NewGetSecretDataUseCase(mockRemote, mockLocalData, mockLocalMeta, tt.sessionType)

			stream, err := uc.Execute(ctx, testMeta)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("ожидалась ошибка %v, но получен nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("получена ошибка: %v, ожидалась: %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("не ожидалось ошибки, но получена: %v", err)
			}

			defer stream.Close()
			res, _ := io.ReadAll(stream)
			if string(res) != tt.wantData {
				t.Errorf("получено %q, ожидалось %q", string(res), tt.wantData)
			}
		})
	}
}
