package orchestrator_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	orchestrator "github.com/SergeyRG/secrets-manager/client/internal/app/orchestrators"
	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	secretsUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/usecases"
	mocks "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/usecases/mocks"
	securityDomain "github.com/SergeyRG/secrets-manager/client/internal/app/security/domain"
	sharedUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/shared/usecases"

	"go.uber.org/mock/gomock"
)

func TestCreateSecretOrch_Execute(t *testing.T) {
	tests := []struct {
		name          string
		sessionType   sharedUsecases.SessionType
		encryptStream securityDomain.StreamEncryptor // Ваш тип из домена security
		mockRepo      func(m *mocks.MockSecretDataRepository)
		inputData     string
		wantEncData   string
		wantErr       error
	}{
		{
			name:        "Успешный сценарий: шифрование потока и создание секрета",
			sessionType: sharedUsecases.SessionTypeRemote,
			encryptStream: func(key []byte, plain io.ReadCloser) (io.ReadCloser, error) {
				defer plain.Close()
				data, err := io.ReadAll(plain)
				if err != nil {
					return nil, err
				}
				// Имитируем шифрование, переводя строку в верхний регистр
				upperData := strings.ToUpper(string(data))
				return io.NopCloser(bytes.NewBufferString(upperData)), nil
			},
			mockRepo: func(m *mocks.MockSecretDataRepository) {
				// Ожидаем, что в репозиторий придет измененный (как бы зашифрованный) поток
				m.EXPECT().
					AddSecretData(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, smd secretsDomain.SecretsMetadata, data io.ReadCloser) error {
						defer data.Close()
						res, err := io.ReadAll(data)
						if err != nil {
							return err
						}
						// Проверяем, что оркестратор передал данные через наш StreamEncryptor
						if string(res) != "MY_RAW_SECRET_DATA" {
							t.Errorf("ожидались зашифрованные данные 'MY_RAW_SECRET_DATA', получено: %q", string(res))
						}
						return nil
					}).
					Times(1)
			},
			inputData: "my_raw_secret_data",
			wantErr:   nil,
		},
		{
			name:        "Ошибка: сбой алгоритма шифрования потока",
			sessionType: sharedUsecases.SessionTypeRemote,
			encryptStream: func(key []byte, plain io.ReadCloser) (io.ReadCloser, error) {
				return nil, errors.New("crypto failure")
			},
			mockRepo: func(m *mocks.MockSecretDataRepository) {
				// До репозитория выполнение дойти не должно
			},
			inputData: "my_raw_secret_data",
			wantErr:   errors.New("crypto failure"),
		},
		{
			name:        "Ошибка: сбой репозитория при отправке зашифрованных данных",
			sessionType: sharedUsecases.SessionTypeRemote,
			encryptStream: func(key []byte, plain io.ReadCloser) (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewBufferString("ENCRYPTED")), nil
			},
			mockRepo: func(m *mocks.MockSecretDataRepository) {
				m.EXPECT().
					AddSecretData(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("network error")).
					Times(1)
			},
			inputData: "my_raw_secret_data",
			wantErr:   errors.New("network error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// 1. Инициализируем изолированный мок репозитория для Use Case
			mockRepo := mocks.NewMockSecretDataRepository(ctrl)
			tt.mockRepo(mockRepo)

			// 2. Создаем реальный Use Case, внедряя мок репозитория
			uc := secretsUsecases.NewCreateSecretUseCase(mockRepo, tt.sessionType)

			// 3. Создаем доменный KeyStorage с тестируемой функцией шифрования потока
			ks := securityDomain.NewKeyStorage(
				nil,              // KeyDecryptor
				nil,              // KeyEncryptor
				tt.encryptStream, // StreamEncryptor (вызывается внутри orch.ks.EncryptData)
				nil,              // StreamDecryptor
			)

			// Инициализируем внутреннее поле key случайными байтами, чтобы обойти ErrKeyEmpty
			if err := ks.GenerateNewKey(); err != nil {
				t.Fatalf("не удалось сгенерировать случайный ключ для KeyStorage: %v", err)
			}

			// 4. Передаем зависимости в оркестратор
			orch := orchestrator.NewCreateSecretOrch(uc, ks)

			// Подготавливаем входящий поток данных
			stream := io.NopCloser(bytes.NewBufferString(tt.inputData))

			// Выполняем оркестратор
			err := orch.Execute(context.Background(), "test_secret", secretsDomain.SecretTypeBinary, stream)

			// 5. Проверяем ошибки
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
		})
	}
}
