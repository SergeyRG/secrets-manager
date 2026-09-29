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

func TestUploadFileUsecase_Execute(t *testing.T) {
	ctx := context.Background()
	dummySrc := io.NopCloser(strings.NewReader("dummy file content"))

	tests := []struct {
		name            string
		src             io.ReadCloser
		mockSetup       func(repo *mocks.MockFilesRepository, stateRepo *mocks.MockFilesStateRepository)
		expectError     bool
		expectedErrText string
	}{
		{
			name: "Успешная загрузка файла",
			src:  dummySrc,
			mockSetup: func(repo *mocks.MockFilesRepository, stateRepo *mocks.MockFilesStateRepository) {
				stateRepo.EXPECT().
					AddFileState(ctx, gomock.Any(), domain.FileStateUploading).
					Return(nil)

				repo.EXPECT().
					AddFile(ctx, gomock.Any(), dummySrc).
					Return(nil)

				stateRepo.EXPECT().
					AddFileState(ctx, gomock.Any(), domain.FileStateReady).
					Return(nil)
			},
			expectError: false,
		},
		{
			name: "Ошибка при начальной записи состояния (Uploading)",
			src:  dummySrc,
			mockSetup: func(repo *mocks.MockFilesRepository, stateRepo *mocks.MockFilesStateRepository) {
				stateRepo.EXPECT().
					AddFileState(ctx, gomock.Any(), domain.FileStateUploading).
					Return(errors.New("db timeout"))
				// Последующие методы не должны вызываться
			},
			expectError:     true,
			expectedErrText: "ошибка записи состояния нового файла",
		},
		{
			name: "Ошибка при загрузке файла в репозиторий",
			src:  dummySrc,
			mockSetup: func(repo *mocks.MockFilesRepository, stateRepo *mocks.MockFilesStateRepository) {
				stateRepo.EXPECT().
					AddFileState(ctx, gomock.Any(), domain.FileStateUploading).
					Return(nil)

				repo.EXPECT().
					AddFile(ctx, gomock.Any(), dummySrc).
					Return(errors.New("s3 upload failed"))
				// Переход в статус Ready не должен вызываться
			},
			expectError:     true,
			expectedErrText: "ошибка загрузки файла на сервер",
		},
		{
			name: "Ошибка при финальной записи состояния (Ready)",
			src:  dummySrc,
			mockSetup: func(repo *mocks.MockFilesRepository, stateRepo *mocks.MockFilesStateRepository) {
				stateRepo.EXPECT().
					AddFileState(ctx, gomock.Any(), domain.FileStateUploading).
					Return(nil)

				repo.EXPECT().
					AddFile(ctx, gomock.Any(), dummySrc).
					Return(nil)

				stateRepo.EXPECT().
					AddFileState(ctx, gomock.Any(), domain.FileStateReady).
					Return(errors.New("db lock"))
			},
			expectError:     true,
			expectedErrText: "успешно загружен, ошибка изменения состояния",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockFilesRepository(ctrl)
			mockStateRepo := mocks.NewMockFilesStateRepository(ctrl)

			tt.mockSetup(mockRepo, mockStateRepo)

			uc := usecases.NewUploadFileUsecase(mockRepo, mockStateRepo)
			fileID, err := uc.Execute(ctx, tt.src)

			if tt.expectError {
				assert.Error(t, err)
				assert.ErrorContains(t, err, tt.expectedErrText)
				assert.Empty(t, fileID)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, fileID)
				assert.IsType(t, domain.FileID(""), fileID)
			}
		})
	}
}
