// get_user_secret_metadata_test.go
package usecases_test

import (
	"context"
	"testing"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	usecases "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases"
	"github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases/mocks"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGetUserSecretMetadataUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	uID := domain.UserID("user-1")

	tests := []struct {
		name          string
		inputVersion  int
		expectVersion int
	}{
		{
			name:          "Позитивная версия без изменений",
			inputVersion:  5,
			expectVersion: 5,
		},
		{
			name:          "Отрицательная версия нормализуется в 0",
			inputVersion:  -10,
			expectVersion: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockSecretMetadataRepository(ctrl)

			mockRepo.EXPECT().
				GetUserSecretMetadataByName(ctx, uID, "my_secret", tt.expectVersion).
				Return(secretsDomain.SecretsMetadata{}, nil)

			uc := usecases.NewGetUserSecretMetadataUseCase(mockRepo)
			_, err := uc.Execute(ctx, uID, "my_secret", tt.inputVersion)

			assert.NoError(t, err)
		})
	}
}
