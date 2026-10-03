package http_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/SergeyRG/secrets-manager/server/internal/app"
	authcontextmanager "github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/auth_context_manager"
	FilesUseCases "github.com/SergeyRG/secrets-manager/server/internal/files/use-cases"
	filesMocks "github.com/SergeyRG/secrets-manager/server/internal/files/use-cases/mocks"
	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	handler "github.com/SergeyRG/secrets-manager/server/internal/secrets/infrastructure/http"
	SecretsUseCases "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases"
	secretsMocks "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases/mocks"
	sharedUseCases "github.com/SergeyRG/secrets-manager/server/internal/shared/use-cases"
)

type StubTxManager struct{}

func (t StubTxManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (t StubTxManager) GetExecutor(ctx context.Context) sharedUseCases.QueryExecutor {
	return nil
}

func TestCreateNewBinarySecretHandler_Handle(t *testing.T) {
	testUserID := domain.UserID("user-777")
	secretName := "my_archive"

	createMultipartRequest := func(t *testing.T, addName bool, addFile bool, contentTypeOverride string) (*http.Request, string) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		if addName {
			err := writer.WriteField("secretName", secretName)
			require.NoError(t, err)
		}
		if addFile {
			part, err := writer.CreateFormFile("file", "test.bin")
			require.NoError(t, err)
			_, err = part.Write([]byte("fake-binary-data"))
			require.NoError(t, err)
		}
		err := writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets/binary", body)

		contentType := writer.FormDataContentType()
		if contentTypeOverride != "" {
			contentType = contentTypeOverride
		}
		req.Header.Set("content-type", contentType)

		return req, writer.Boundary()
	}

	t.Run("Успешное создание бинарного секрета", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFilesRepo := filesMocks.NewMockFilesRepository(ctrl)
		mockFilesStateRepo := filesMocks.NewMockFilesStateRepository(ctrl)
		mockSecretMetaRepo := secretsMocks.NewMockSecretMetadataRepository(ctrl)
		mockSecretDataSaver := secretsMocks.NewMockSecretDataSaver(ctrl)

		mockFilesRepo.EXPECT().AddFile(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		mockFilesRepo.EXPECT().GetFile(gomock.Any(), gomock.Any()).Return(io.NopCloser(strings.NewReader("data")), nil).Times(1)
		mockFilesStateRepo.EXPECT().AddFileState(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockSecretMetaRepo.EXPECT().AddSecretMetadata(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		mockSecretDataSaver.EXPECT().SaveSecretData(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

		saversMap := map[secretsDomain.SecretType]SecretsUseCases.SecretDataSaver{
			secretsDomain.SecretTypeBinary: mockSecretDataSaver,
		}

		uploadUC := FilesUseCases.NewUploadFileUsecase(mockFilesRepo, mockFilesStateRepo)
		getFileUC := FilesUseCases.NewGetFileUsecase(mockFilesRepo)
		createSecretUC := SecretsUseCases.NewCreateNewSecretUseCase(mockSecretMetaRepo, saversMap, StubTxManager{}, zap.NewNop())

		// ИСПРАВЛЕНО: Изменено имя конструктора на правильное NewCreateNewSecretOrch
		orch := app.NewCreateNewSecretOrch(uploadUC, getFileUC, createSecretUC)

		h := handler.NewCreateNewBinarySecretHandler(orch)

		req, _ := createMultipartRequest(t, true, true, "")

		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)

		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusCreated, rw.Code)
	})

	t.Run("Ошибка: Пользователь не авторизован (отсутствует UserID в контексте)", func(t *testing.T) {
		h := handler.NewCreateNewBinarySecretHandler(nil)

		req, _ := createMultipartRequest(t, true, true, "")
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusUnauthorized, rw.Code)
	})

	t.Run("Ошибка: Невалидный Content-Type (не multipart/form-data)", func(t *testing.T) {
		h := handler.NewCreateNewBinarySecretHandler(nil)

		req, _ := createMultipartRequest(t, true, true, "application/json")
		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})

	t.Run("Ошибка: Падение оркестратора при загрузке или сохранении", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFilesRepo := filesMocks.NewMockFilesRepository(ctrl)
		mockFilesStateRepo := filesMocks.NewMockFilesStateRepository(ctrl)

		mockFilesRepo.EXPECT().AddFile(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("disk failure")).Times(1)
		mockFilesStateRepo.EXPECT().AddFileState(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		uploadUC := FilesUseCases.NewUploadFileUsecase(mockFilesRepo, mockFilesStateRepo)

		orch := app.NewCreateNewSecretOrch(uploadUC, nil, nil)

		h := handler.NewCreateNewBinarySecretHandler(orch)

		req, _ := createMultipartRequest(t, true, true, "")
		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusInternalServerError, rw.Code)
		assert.Contains(t, rw.Body.String(), "ошибка создания бинарного секрета")
	})
}
