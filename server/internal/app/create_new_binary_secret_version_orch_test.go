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

type StubTransactionManagerVersion struct{}

func (t StubTransactionManagerVersion) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (t StubTransactionManagerVersion) GetExecutor(ctx context.Context) sharedUseCases.QueryExecutor {
	return nil
}

func TestCreateNewBinarySecretVersionOrch_Execute(t *testing.T) {
	userID := domain.UserID("user-123")
	secretName := "archive_tar_gz"

	expectedMetadata := secretsDomain.SecretsMetadata{
		SecretID:   domain.SecretID("secret-uuid"),
		SecretName: secretName,
		Version:    2,
	}

	t.Run("Успешный сценарий оркестрации новой версии", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFilesRepo := filesMocks.NewMockFilesRepository(ctrl)
		mockFilesStateRepo := filesMocks.NewMockFilesStateRepository(ctrl)
		mockSecretMetaRepo := secretsMocks.NewMockSecretMetadataRepository(ctrl)
		mockSecretDataSaver := secretsMocks.NewMockSecretDataSaver(ctrl)

		mockFileStream := io.NopCloser(strings.NewReader("updated-file-content"))

		// --- Ожидания для домена Files ---
		mockFilesRepo.EXPECT().AddFile(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		mockFilesRepo.EXPECT().GetFile(gomock.Any(), gomock.Any()).Return(mockFileStream, nil).Times(1)

		mockFilesStateRepo.EXPECT().
			AddFileState(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil).
			AnyTimes()

		// --- Ожидания для домена Secrets ---
		// ИСПРАВЛЕНО: Явно передаем SecretTypeBinary, чтобы UseCase мог найти сейвер в saversMap
		mockSecretMetaRepo.EXPECT().
			GetUserSecretMetadataByName(gomock.Any(), userID, secretName, gomock.Any()).
			Return(secretsDomain.SecretsMetadata{
				SecretID:   "secret-uuid",
				SecretName: secretName,
				SecretType: secretsDomain.SecretTypeBinary, // Критично для мапы сейверов!
				Version:    1,
			}, nil).
			AnyTimes()

		mockSecretMetaRepo.EXPECT().AddSecretMetadata(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		mockSecretDataSaver.EXPECT().SaveSecretData(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

		saversMap := map[secretsDomain.SecretType]SecretsUseCases.SecretDataSaver{
			secretsDomain.SecretTypeBinary: mockSecretDataSaver,
		}

		uploadUC := FilesUseCases.NewUploadFileUsecase(mockFilesRepo, mockFilesStateRepo)
		getFileUC := FilesUseCases.NewGetFileUsecase(mockFilesRepo)
		addSecretVersionUC := SecretsUseCases.NewAddSecretVersionUseCase(
			mockSecretMetaRepo,
			saversMap,
			StubTransactionManagerVersion{},
			zap.NewNop(),
		)

		orch := app.NewCreateNewBinarySecretVersionOrch(uploadUC, getFileUC, addSecretVersionUC)

		src := io.NopCloser(strings.NewReader("new-binary-version-data"))
		res, err := orch.Execute(context.Background(), userID, secretName, src)

		assert.NoError(t, err)
		assert.Equal(t, expectedMetadata.SecretName, res.SecretName)
	})

	t.Run("Ошибка на этапе загрузки файла (Upload)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFilesRepo := filesMocks.NewMockFilesRepository(ctrl)
		mockFilesStateRepo := filesMocks.NewMockFilesStateRepository(ctrl)

		// ИСПРАВЛЕНО: Добавляем AddFileState, так как UseCase вызывает его до или во время AddFile
		mockFilesStateRepo.EXPECT().
			AddFileState(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil).
			AnyTimes()

		mockFilesRepo.EXPECT().
			AddFile(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(errors.New("upload failed")).
			Times(1)

		uploadUC := FilesUseCases.NewUploadFileUsecase(mockFilesRepo, mockFilesStateRepo)
		orch := app.NewCreateNewBinarySecretVersionOrch(uploadUC, nil, nil)

		src := io.NopCloser(strings.NewReader("data"))
		res, err := orch.Execute(context.Background(), userID, secretName, src)

		assert.Empty(t, res)
		assert.EqualError(t, err, "ошибка загрузки файла")
	})

	t.Run("Ошибка на этапе открытия сохраненного файла (GetFile)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFilesRepo := filesMocks.NewMockFilesRepository(ctrl)
		mockFilesStateRepo := filesMocks.NewMockFilesStateRepository(ctrl)

		mockFilesRepo.EXPECT().AddFile(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		mockFilesStateRepo.EXPECT().AddFileState(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		mockFilesRepo.EXPECT().
			GetFile(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("read error")).
			Times(1)

		uploadUC := FilesUseCases.NewUploadFileUsecase(mockFilesRepo, mockFilesStateRepo)
		getFileUC := FilesUseCases.NewGetFileUsecase(mockFilesRepo)
		orch := app.NewCreateNewBinarySecretVersionOrch(uploadUC, getFileUC, nil)

		src := io.NopCloser(strings.NewReader("data"))
		res, err := orch.Execute(context.Background(), userID, secretName, src)

		assert.Empty(t, res)
		assert.EqualError(t, err, "ошибка загрузки файла")
	})

	t.Run("Ошибка на этапе добавления версии секрета (AddSecretVersion)", func(t *testing.T) {
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
			GetUserSecretMetadataByName(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(secretsDomain.SecretsMetadata{}, errors.New("secret not found")).
			Times(1)

		saversMap := map[secretsDomain.SecretType]SecretsUseCases.SecretDataSaver{
			secretsDomain.SecretTypeBinary: mockSecretDataSaver,
		}

		uploadUC := FilesUseCases.NewUploadFileUsecase(mockFilesRepo, mockFilesStateRepo)
		getFileUC := FilesUseCases.NewGetFileUsecase(mockFilesRepo)
		addSecretVersionUC := SecretsUseCases.NewAddSecretVersionUseCase(
			mockSecretMetaRepo,
			saversMap,
			StubTransactionManagerVersion{},
			zap.NewNop(),
		)

		orch := app.NewCreateNewBinarySecretVersionOrch(uploadUC, getFileUC, addSecretVersionUC)

		src := io.NopCloser(strings.NewReader("data"))
		res, err := orch.Execute(context.Background(), userID, secretName, src)

		assert.Empty(t, res)
		assert.EqualError(t, err, "Ошибка создания бинарного секрета")
	})
}
