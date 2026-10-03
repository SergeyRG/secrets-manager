// get_user_secret_metadata_page_test.go
package usecases_test

import (
	"context"
	"testing"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	usecases "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases"
	"github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases/mocks"
	sharedMocks "github.com/SergeyRG/secrets-manager/server/internal/shared/use-cases/mocks"

	"go.uber.org/mock/gomock"
)

func TestGetUserSecretMetadataPageUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	uID := domain.UserID("user-1")

	tests := []struct {
		name          string
		inputPage     int
		inputPerPage  int
		expectPage    int
		expectPerPage int
	}{
		{
			name:          "Корректные значения",
			inputPage:     2,
			inputPerPage:  10,
			expectPage:    2,
			expectPerPage: 10,
		},
		{
			name:          "Отрицательная страница нормализуется в 1",
			inputPage:     -5,
			inputPerPage:  10,
			expectPage:    1,
			expectPerPage: 10,
		},
		{
			name:          "Превышение лимита нормализуется в 20",
			inputPage:     1,
			inputPerPage:  100,
			expectPage:    1,
			expectPerPage: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockSecretMetadataRepository(ctrl)
			mockTxm := sharedMocks.NewMockTransactionManager(ctrl)

			mockRepo.EXPECT().
				GetUserSecretsMetadataPage(ctx, uID, tt.expectPage, tt.expectPerPage).
				Return(nil)

			uc := usecases.NewGetUserSecretMetadataPageUseCase(mockRepo, mockTxm)
			uc.Execute(ctx, uID, tt.inputPage, tt.inputPerPage)
		})
	}
}
