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

type StubTxManagerVersionHandler struct{}

func (t StubTxManagerVersionHandler) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (t StubTxManagerVersionHandler) GetExecutor(ctx context.Context) sharedUseCases.QueryExecutor {
	return nil
}

func TestCreateNewBinarySecretVersionHandler_Handle(t *testing.T) {
	testUserID := domain.UserID("user-777")
	secretName := "my_archive_version"

	createMultipartRequestVersion := func(t *testing.T, addName bool, addFile bool, contentTypeOverride string) *http.Request {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		if addName {
			err := writer.WriteField("secretName", secretName)
			require.NoError(t, err)
		}
		if addFile {
			part, err := writer.CreateFormFile("file", "update.bin")
			require.NoError(t, err)
			_, err = part.Write([]byte("updated-binary-content-bytes"))
			require.NoError(t, err)
		}
		err := writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets/binary/version", body)

		contentType := writer.FormDataContentType()
		if contentTypeOverride != "" {
			contentType = contentTypeOverride
		}
		req.Header.Set("content-type", contentType)

		return req
	}

	t.Run("Успешное создание новой версии бинарного секрета", func(t *testing.T) {
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
			GetUserSecretMetadataByName(gomock.Any(), testUserID, secretName, gomock.Any()).
			Return(secretsDomain.SecretsMetadata{
				SecretID:   "secret-uuid-111",
				SecretName: secretName,
				SecretType: secretsDomain.SecretTypeBinary,
				Version:    1,
			}, nil).
			Times(1)

		mockSecretMetaRepo.EXPECT().AddSecretMetadata(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		mockSecretDataSaver.EXPECT().SaveSecretData(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

		saversMap := map[secretsDomain.SecretType]SecretsUseCases.SecretDataSaver{
			secretsDomain.SecretTypeBinary: mockSecretDataSaver,
		}

		uploadUC := FilesUseCases.NewUploadFileUsecase(mockFilesRepo, mockFilesStateRepo)
		getFileUC := FilesUseCases.NewGetFileUsecase(mockFilesRepo)
		addSecretVersionUC := SecretsUseCases.NewAddSecretVersionUseCase(mockSecretMetaRepo, saversMap, StubTxManagerVersionHandler{}, zap.NewNop())

		orch := app.NewCreateNewBinarySecretVersionOrch(uploadUC, getFileUC, addSecretVersionUC)

		h := handler.NewCreateNewBinarySecretVersionHandler(orch)

		req := createMultipartRequestVersion(t, true, true, "")

		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)

		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusCreated, rw.Code)
	})

	t.Run("Ошибка: Неавторизованный запрос (нет UserID в контексте)", func(t *testing.T) {
		h := handler.NewCreateNewBinarySecretVersionHandler(nil)

		req := createMultipartRequestVersion(t, true, true, "")
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusUnauthorized, rw.Code)
	})

	t.Run("Ошибка: Неверный тип Content-Type", func(t *testing.T) {
		h := handler.NewCreateNewBinarySecretVersionHandler(nil)

		req := createMultipartRequestVersion(t, true, true, "application/octet-stream")
		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})

	t.Run("Ошибка: Сбой оркестратора при добавлении новой версии", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFilesRepo := filesMocks.NewMockFilesRepository(ctrl)
		mockFilesStateRepo := filesMocks.NewMockFilesStateRepository(ctrl)

		mockFilesRepo.EXPECT().AddFile(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("disk full")).Times(1)
		mockFilesStateRepo.EXPECT().AddFileState(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		uploadUC := FilesUseCases.NewUploadFileUsecase(mockFilesRepo, mockFilesStateRepo)
		orch := app.NewCreateNewBinarySecretVersionOrch(uploadUC, nil, nil)

		h := handler.NewCreateNewBinarySecretVersionHandler(orch)

		req := createMultipartRequestVersion(t, true, true, "")
		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusInternalServerError, rw.Code)
		assert.Contains(t, rw.Body.String(), "ошибка создания бинарного секрета")
	})
}
