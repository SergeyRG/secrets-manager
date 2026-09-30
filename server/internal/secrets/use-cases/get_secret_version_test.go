// get_secret_version_test.go
package usecases_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	usecases "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases"
	"github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases/mocks"
	sharedMocks "github.com/SergeyRG/secrets-manager/server/internal/shared/use-cases/mocks"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestGetSecretVersionUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	uID := domain.UserID("user-1")
	validSecretType := secretsDomain.SecretTypeBinary
	dummyReader := io.NopCloser(strings.NewReader("secret content"))

	testSm := secretsDomain.SecretsMetadata{
		UserID:     uID,
		SecretName: "my_secret", // Исправлено имя[cite: 24]
		SecretType: validSecretType,
		Version:    1,
	}

	tests := []struct {
		name          string
		version       int
		mockSetup     func(repo *mocks.MockSecretMetadataRepository, receiver *mocks.MockSecretDataReceiver)
		expectedError error
	}{
		{
			name:    "Успешное получение версии",
			version: 1,
			mockSetup: func(repo *mocks.MockSecretMetadataRepository, receiver *mocks.MockSecretDataReceiver) {
				repo.EXPECT().GetUserSecretMetadataByName(ctx, uID, "my_secret", 1).Return(testSm, nil)
				receiver.EXPECT().ReceiveSecretData(ctx, testSm).Return(dummyReader, nil)
			},
			expectedError: nil,
		},
		{
			name:    "Метаданные не найдены",
			version: 1,
			mockSetup: func(repo *mocks.MockSecretMetadataRepository, receiver *mocks.MockSecretDataReceiver) {
				repo.EXPECT().GetUserSecretMetadataByName(ctx, uID, "my_secret", 1).Return(secretsDomain.SecretsMetadata{}, errors.New("not found"))
			},
			expectedError: errors.New("not found"),
		},
		{
			name:    "Отсутствует Receiver для типа",
			version: 1,
			mockSetup: func(repo *mocks.MockSecretMetadataRepository, receiver *mocks.MockSecretDataReceiver) {
				invalidSm := testSm
				invalidSm.SecretType = secretsDomain.SecretTypeAuthData
				repo.EXPECT().GetUserSecretMetadataByName(ctx, uID, "my_secret", 1).Return(invalidSm, nil)
			},
			expectedError: errors.New("неизвестный тип секрета"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockSecretMetadataRepository(ctrl)
			mockReceiver := mocks.NewMockSecretDataReceiver(ctrl)

			mockTxm := sharedMocks.NewMockTransactionManager(ctrl)

			receivers := map[secretsDomain.SecretType]usecases.SecretDataReceiver{
				validSecretType: mockReceiver,
			}

			tt.mockSetup(mockRepo, mockReceiver)

			uc := usecases.NewGetSecretVersionUseCase(mockRepo, receivers, mockTxm, zap.NewNop())
			data, err := uc.Execute(ctx, uID, "my_secret", tt.version)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
				assert.Nil(t, data)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, dummyReader, data)
			}
		})
	}
}
