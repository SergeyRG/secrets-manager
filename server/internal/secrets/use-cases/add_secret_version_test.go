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

func TestAddSecretVersionUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	uID := domain.UserID("user-1")
	secretType := secretsDomain.SecretType(1)
	dummyReader := io.NopCloser(strings.NewReader("new secret data"))

	curSm := secretsDomain.SecretsMetadata{
		UserID:     uID,
		SecretName: "my-secret",
		SecretType: secretType,
		Version:    1,
	}

	tests := []struct {
		name        string
		mockSetup   func(repo *mocks.MockSecretMetadataRepository, saver *mocks.MockSecretDataSaver)
		expectError bool
	}{
		{
			name: "Успешное добавление новой версии секрета",
			mockSetup: func(repo *mocks.MockSecretMetadataRepository, saver *mocks.MockSecretDataSaver) {
				repo.EXPECT().GetUserSecretMetadataByName(ctx, uID, "my-secret", 0).Return(curSm, nil)
				repo.EXPECT().AddSecretMetadata(ctx, gomock.Any()).Return(nil)
				saver.EXPECT().SaveSecretData(ctx, gomock.Any(), dummyReader).Return(nil)
			},
			expectError: false,
		},
		{
			name: "Ошибка получения предыдущей версии из БД",
			mockSetup: func(repo *mocks.MockSecretMetadataRepository, saver *mocks.MockSecretDataSaver) {
				repo.EXPECT().GetUserSecretMetadataByName(ctx, uID, "my-secret", 0).Return(secretsDomain.SecretsMetadata{}, errors.New("not found"))
			},
			expectError: true,
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
				secretType: mockSaver,
			}

			tt.mockSetup(mockRepo, mockSaver)

			uc := usecases.NewAddSecretVersionUseCase(mockRepo, savers, mockTxm, zap.NewNop())
			newSm, err := uc.Execute(ctx, uID, "my-secret", dummyReader)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, int64(2), newSm.Version)
			}
		})
	}
}
