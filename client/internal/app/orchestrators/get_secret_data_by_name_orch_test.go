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

func TestGetSecretDataByNameOrch_Execute(t *testing.T) {
	tests := []struct {
		name          string
		sessionType   sharedUsecases.SessionType
		inputName     string // Динамическое имя секрета для каждого тест-кейса
		inputVersion  int
		decryptStream securityDomain.StreamDecryptor
		mockMetadata  func(m *mocks.MockSecretMetadataRepository)
		mockData      func(m *mocks.MockSecretDataRepository)
		wantMeta      secretsDomain.SecretsMetadata
		wantData      string
		wantErr       error
	}{
		{
			name:         "Успешное получение и расшифрование секрета",
			sessionType:  sharedUsecases.SessionTypeRemote,
			inputName:    "my_secret",
			inputVersion: 1,
			decryptStream: func(key []byte, encrypted io.ReadCloser) (io.ReadCloser, error) {
				defer encrypted.Close()
				data, _ := io.ReadAll(encrypted)
				lowerData := strings.ToLower(string(data))
				return io.NopCloser(bytes.NewBufferString(lowerData)), nil
			},
			mockMetadata: func(m *mocks.MockSecretMetadataRepository) {
				m.EXPECT().
					GetUserSecretMetadataByName(gomock.Any(), "my_secret", 1).
					Return(secretsDomain.SecretsMetadata{
						SecretName: "my_secret",
						SecretType: secretsDomain.SecretTypeFreeText,
						Version:    1,
					}, nil).
					AnyTimes()
			},
			mockData: func(m *mocks.MockSecretDataRepository) {
				encryptedStream := io.NopCloser(bytes.NewBufferString("ENCRYPTED_SECRET_PAYLOAD"))
				m.EXPECT().
					GetSecretData(gomock.Any(), gomock.Any()).
					Return(encryptedStream, nil).
					Times(1)
			},
			wantMeta: secretsDomain.SecretsMetadata{
				SecretName: "my_secret",
				SecretType: secretsDomain.SecretTypeFreeText,
				Version:    1,
			},
			wantData: "encrypted_secret_payload",
			wantErr:  nil,
		},
		{
			name:         "Ошибка: Метаданные секрета не найдены",
			sessionType:  sharedUsecases.SessionTypeRemote,
			inputName:    "missing_secret",
			inputVersion: 0,
			decryptStream: func(key []byte, encrypted io.ReadCloser) (io.ReadCloser, error) {
				return nil, nil
			},
			mockMetadata: func(m *mocks.MockSecretMetadataRepository) {
				m.EXPECT().
					GetUserSecretMetadataByName(gomock.Any(), "missing_secret", 0).
					Return(secretsDomain.SecretsMetadata{}, errors.New("not found")).
					AnyTimes() // Устойчиво к повторным вызовам внутри цепочки выполнения
			},
			mockData: func(m *mocks.MockSecretDataRepository) {

			},
			wantMeta: secretsDomain.SecretsMetadata{},
			wantData: "",
			wantErr:  errors.New("not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockMetaRepo := mocks.NewMockSecretMetadataRepository(ctrl)
			mockDataRepo := mocks.NewMockSecretDataRepository(ctrl)

			tt.mockMetadata(mockMetaRepo)
			tt.mockData(mockDataRepo)

			getMetaUC := secretsUsecases.NewGetSecretMetadataUseCase(mockMetaRepo, mockMetaRepo, tt.sessionType)
			getDataUC := secretsUsecases.NewGetSecretDataUseCase(mockDataRepo, mockDataRepo, mockMetaRepo, tt.sessionType)

			ks := securityDomain.NewKeyStorage(
				nil,
				nil,
				nil,
				tt.decryptStream,
			)

			if err := ks.GenerateNewKey(); err != nil {
				t.Fatalf("не удалось сгенерировать тестовый ключ: %v", err)
			}

			orch := orchestrator.NewGetSecretDataByNameOrch(getMetaUC, getDataUC, ks)

			meta, decryptedStream, err := orch.Execute(context.Background(), tt.inputName, tt.inputVersion)

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

			if meta.SecretName != tt.wantMeta.SecretName || meta.SecretType != tt.wantMeta.SecretType || meta.Version != tt.wantMeta.Version {
				t.Errorf("получены неверные метаданные: %+v, ожидалось: %+v", meta, tt.wantMeta)
			}

			defer decryptedStream.Close()
			res, err := io.ReadAll(decryptedStream)
			if err != nil {
				t.Fatalf("ошибка чтения из расшифрованного потока: %v", err)
			}
			if string(res) != tt.wantData {
				t.Errorf("получено: %q, ожидалось: %q", string(res), tt.wantData)
			}
		})
	}
}
