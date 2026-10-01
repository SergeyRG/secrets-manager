package app_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/SergeyRG/secrets-manager/server/internal/app"
	FilesUseCases "github.com/SergeyRG/secrets-manager/server/internal/files/use-cases"
	filesMocks "github.com/SergeyRG/secrets-manager/server/internal/files/use-cases/mocks"
	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	SecretsUseCases "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases"
	secretsMocks "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases/mocks"
	sharedUseCases "github.com/SergeyRG/secrets-manager/server/internal/shared/use-cases"
)

type StubTransactionManager struct{}

func (t StubTransactionManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (t StubTransactionManager) GetExecutor(ctx context.Context) sharedUseCases.QueryExecutor {
	return nil
}

func TestCreateNewBinarySecretOrch_Execute(t *testing.T) {
	userID := domain.UserID("user-123")
	secretName := "archive_tar_gz"

	expectedMetadata := secretsDomain.SecretsMetadata{
		SecretID:   domain.SecretID("secret-uuid"),
		SecretName: secretName,
		Version:    1,
	}

	t.Run("Успешный сценарий оркестрации", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFilesRepo := filesMocks.NewMockFilesRepository(ctrl)
		mockFilesStateRepo := filesMocks.NewMockFilesStateRepository(ctrl)
		mockSecretMetaRepo := secretsMocks.NewMockSecretMetadataRepository(ctrl)
		mockSecretDataSaver := secretsMocks.NewMockSecretDataSaver(ctrl)

		mockFileStream := io.NopCloser(strings.NewReader("file-content"))

		mockFilesRepo.EXPECT().
			AddFile(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil).
			Times(1)

		mockFilesRepo.EXPECT().
			GetFile(gomock.Any(), gomock.Any()).
			Return(mockFileStream, nil).
			Times(1)

		mockFilesStateRepo.EXPECT().
			AddFileState(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil).
			AnyTimes()

		mockSecretMetaRepo.EXPECT().
			AddSecretMetadata(gomock.Any(), gomock.Any()).
			Return(nil).
			Times(1)

		mockSecretDataSaver.EXPECT().
			SaveSecretData(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil).
			Times(1)

		saversMap := map[secretsDomain.SecretType]SecretsUseCases.SecretDataSaver{
			secretsDomain.SecretTypeBinary: mockSecretDataSaver,
		}

		uploadUC := FilesUseCases.NewUploadFileUsecase(mockFilesRepo, mockFilesStateRepo)
		getFileUC := FilesUseCases.NewGetFileUsecase(mockFilesRepo)
		createSecretUC := SecretsUseCases.NewCreateNewSecretUseCase(
			mockSecretMetaRepo,
			saversMap,
			StubTransactionManager{},
			zap.NewNop(),
		)

		orch := app.NewCreateNewSecretOrch(uploadUC, getFileUC, createSecretUC)

		src := io.NopCloser(strings.NewReader("raw-binary-data"))
		res, err := orch.Execute(context.Background(), userID, secretName, src)

		assert.NoError(t, err)
		assert.Equal(t, expectedMetadata.SecretName, res.SecretName)
	})

	t.Run("Ошибка на этапе сохранения секрета (CreateNewSecret)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFilesRepo := filesMocks.NewMockFilesRepository(ctrl)
		mockFilesStateRepo := filesMocks.NewMockFilesStateRepository(ctrl)
		mockSecretMetaRepo := secretsMocks.NewMockSecretMetadataRepository(ctrl)
		mockSecretDataSaver := secretsMocks.NewMockSecretDataSaver(ctrl)

		mockFilesRepo.EXPECT().AddFile(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		mockFilesRepo.EXPECT().GetFile(gomock.Any(), gomock.Any()).Return(io.NopCloser(strings.NewReader("data")), nil).Times(1)
		mockFilesStateRepo.EXPECT().AddFileState(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		mockSecretMetaRepo.EXPECT().
			AddSecretMetadata(gomock.Any(), gomock.Any()).
			Return(errors.New("db error")).
			Times(1)

		saversMap := map[secretsDomain.SecretType]SecretsUseCases.SecretDataSaver{
			secretsDomain.SecretTypeBinary: mockSecretDataSaver,
		}

		uploadUC := FilesUseCases.NewUploadFileUsecase(mockFilesRepo, mockFilesStateRepo)
		getFileUC := FilesUseCases.NewGetFileUsecase(mockFilesRepo)
		createSecretUC := SecretsUseCases.NewCreateNewSecretUseCase(
			mockSecretMetaRepo,
			saversMap,
			StubTransactionManager{},
			zap.NewNop(),
		)

		orch := app.NewCreateNewSecretOrch(uploadUC, getFileUC, createSecretUC)

		src := io.NopCloser(strings.NewReader("raw-binary-data"))
		res, err := orch.Execute(context.Background(), userID, secretName, src)

		assert.Empty(t, res)
		assert.EqualError(t, err, "Ошибка создания бинарного секрета")
	})
}
