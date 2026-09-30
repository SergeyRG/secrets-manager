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

func TestCreateNewSecretUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	uID := domain.UserID("user-1")
	validSecretType := secretsDomain.SecretTypeBinary
	dummyReader := io.NopCloser(strings.NewReader("secret data"))

	tests := []struct {
		name          string
		secretType    secretsDomain.SecretType
		mockSetup     func(repo *mocks.MockSecretMetadataRepository, saver *mocks.MockSecretDataSaver)
		expectedError error
	}{
		{
			name:       "Успешное создание секрета",
			secretType: validSecretType,
			mockSetup: func(repo *mocks.MockSecretMetadataRepository, saver *mocks.MockSecretDataSaver) {
				repo.EXPECT().AddSecretMetadata(ctx, gomock.Any()).Return(nil)
				saver.EXPECT().SaveSecretData(ctx, gomock.Any(), dummyReader).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:       "Ошибка репозитория метаданных",
			secretType: validSecretType,
			mockSetup: func(repo *mocks.MockSecretMetadataRepository, saver *mocks.MockSecretDataSaver) {
				repo.EXPECT().AddSecretMetadata(ctx, gomock.Any()).Return(errors.New("db error"))
			},
			expectedError: errors.New("db error"),
		},
		{
			name:       "Отсутствует Saver для типа",
			secretType: secretsDomain.SecretTypeAuthData, // Валидный тип для домена, но отсутствует в мапе savers
			mockSetup: func(repo *mocks.MockSecretMetadataRepository, saver *mocks.MockSecretDataSaver) {
				repo.EXPECT().AddSecretMetadata(ctx, gomock.Any()).Return(nil)
			},
			expectedError: errors.New("неизвестный тип секрета"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockSecretMetadataRepository(ctrl)
			mockSaver := mocks.NewMockSecretDataSaver(ctrl)
			mockTxm := sharedMocks.NewMockTransactionManager(ctrl)

			mockTxm.EXPECT().
				WithinTransaction(gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, fn func(txCtx context.Context) error) error {
					return fn(ctx)
				}).AnyTimes()

			savers := map[secretsDomain.SecretType]usecases.SecretDataSaver{
				validSecretType: mockSaver,
			}

			tt.mockSetup(mockRepo, mockSaver)

			uc := usecases.NewCreateNewSecretUseCase(mockRepo, savers, mockTxm, zap.NewNop())

			sm, err := uc.Execute(ctx, uID, "my_secret", tt.secretType, dummyReader)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, sm.SecretID)
			}
		})
	}
}
