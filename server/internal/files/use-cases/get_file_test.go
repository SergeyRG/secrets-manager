package usecases_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/SergeyRG/secrets-manager/server/internal/files/domain"
	usecases "github.com/SergeyRG/secrets-manager/server/internal/files/use-cases"
	"github.com/SergeyRG/secrets-manager/server/internal/files/use-cases/mocks"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGetFileUsecase_Execute(t *testing.T) {
	ctx := context.Background()
	fileID := domain.FileID("test-file-id")

	dummyContent := "file content"
	dummyReadCloser := io.NopCloser(strings.NewReader(dummyContent))

	tests := []struct {
		name            string
		fileID          domain.FileID
		mockSetup       func(repo *mocks.MockFilesRepository)
		expectError     bool
		expectedErrText string
		expectedResult  io.ReadCloser
	}{
		{
			name:   "Успешное получение файла",
			fileID: fileID,
			mockSetup: func(repo *mocks.MockFilesRepository) {
				repo.EXPECT().GetFile(ctx, fileID).Return(dummyReadCloser, nil)
			},
			expectError:    false,
			expectedResult: dummyReadCloser,
		},
		{
			name:   "Ошибка получения файла из репозитория",
			fileID: fileID,
			mockSetup: func(repo *mocks.MockFilesRepository) {
				repo.EXPECT().GetFile(ctx, fileID).Return(nil, errors.New("storage offline"))
			},
			expectError:     true,
			expectedErrText: "ошибка получения файла: storage offline",
			expectedResult:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockFilesRepository(ctrl)
			tt.mockSetup(mockRepo)

			uc := usecases.NewGetFileUsecase(mockRepo)
			result, err := uc.Execute(ctx, tt.fileID)

			if tt.expectError {
				assert.Error(t, err)
				assert.ErrorContains(t, err, tt.expectedErrText)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}
