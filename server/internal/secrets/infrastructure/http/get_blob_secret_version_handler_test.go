package http_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	authcontextmanager "github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/auth_context_manager"
	secretDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	handler "github.com/SergeyRG/secrets-manager/server/internal/secrets/infrastructure/http"
	secretsUseCases "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases"
	secretsMocks "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases/mocks"
	sharedUseCases "github.com/SergeyRG/secrets-manager/server/internal/shared/use-cases"
)

type StubTxManagerGetVersionHandler struct{}

func (t StubTxManagerGetVersionHandler) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (t StubTxManagerGetVersionHandler) GetExecutor(ctx context.Context) sharedUseCases.QueryExecutor {
	return nil
}

func TestGetBlobSecretVersionHandler_Handle(t *testing.T) {
	testUserID := domain.UserID("user-abc-123")
	secretName := "my_backup_file"
	versionStr := "3"
	versionInt := 3
	secretDataPayload := "binary-encrypted-blob-stream-content"

	t.Run("Успешное получение и стриминг бинарного секрета", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSecretMetaRepo := secretsMocks.NewMockSecretMetadataRepository(ctrl)
		mockSecretReceiver := secretsMocks.NewMockSecretDataReceiver(ctrl)

		mockSecretMetaRepo.EXPECT().
			GetUserSecretMetadataByName(gomock.Any(), testUserID, secretName, versionInt).
			Return(secretDomain.SecretsMetadata{
				UserID:     testUserID,
				SecretID:   "secret-uuid",
				SecretName: secretName,
				SecretType: secretDomain.SecretTypeBinary,
				Version:    int64(versionInt),
				VersionID:  "version-uuid-3",
			}, nil).
			Times(1)

		mockStream := io.NopCloser(strings.NewReader(secretDataPayload))
		mockSecretReceiver.EXPECT().
			ReceiveSecretData(gomock.Any(), gomock.Any()).
			Return(mockStream, nil).
			Times(1)

		receiversMap := map[secretDomain.SecretType]secretsUseCases.SecretDataReceiver{
			secretDomain.SecretTypeBinary: mockSecretReceiver,
		}

		getSecretVersionUC := secretsUseCases.NewGetSecretVersionUseCase(
			mockSecretMetaRepo,
			receiversMap,
			StubTxManagerGetVersionHandler{},
			zap.NewNop(),
		)
		h := handler.NewGetBlobSecretVersionHandler(getSecretVersionUC)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/binary/version", nil)
		req.Header.Set("X-Secret-Name", secretName)
		req.Header.Set("X-Secret-Version", versionStr)

		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)

		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusOK, rw.Code)
		assert.Equal(t, "application/octet-stream", rw.Header().Get("Content-Type"))
		assert.Equal(t, `inline; filename="`+secretName+`.enc"`, rw.Header().Get("Content-Disposition"))
		assert.Equal(t, "none", rw.Header().Get("Accept-Ranges"))
		assert.Equal(t, secretDataPayload, rw.Body.String())
	})

	t.Run("Ошибка: Запрос не авторизован (нет UserID в контексте)", func(t *testing.T) {
		h := handler.NewGetBlobSecretVersionHandler(nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/binary/version", nil)
		req.Header.Set("X-Secret-Name", secretName)
		req.Header.Set("X-Secret-Version", versionStr)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusUnauthorized, rw.Code)
	})

	t.Run("Ошибка: Не заданы необходимые заголовки (X-Secret-Name)", func(t *testing.T) {
		h := handler.NewGetBlobSecretVersionHandler(nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/binary/version", nil)
		req.Header.Set("X-Secret-Version", versionStr)

		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
		assert.Contains(t, rw.Body.String(), "не заданы необходимые заголовки")
	})

	t.Run("Ошибка: Версия в заголовке не является целым числом", func(t *testing.T) {
		h := handler.NewGetBlobSecretVersionHandler(nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/binary/version", nil)
		req.Header.Set("X-Secret-Name", secretName)
		req.Header.Set("X-Secret-Version", "not-a-number")

		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
		assert.Contains(t, rw.Body.String(), "версия должна быть целым числом")
	})

	t.Run("Ошибка: Сбой UseCase (например, секрет не найден в БД)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSecretMetaRepo := secretsMocks.NewMockSecretMetadataRepository(ctrl)

		mockSecretMetaRepo.EXPECT().
			GetUserSecretMetadataByName(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(secretDomain.SecretsMetadata{}, errors.New("sql: no rows in result set")).
			Times(1)

		getSecretVersionUC := secretsUseCases.NewGetSecretVersionUseCase(
			mockSecretMetaRepo,
			nil,
			StubTxManagerGetVersionHandler{},
			zap.NewNop(),
		)
		h := handler.NewGetBlobSecretVersionHandler(getSecretVersionUC)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/binary/version", nil)
		req.Header.Set("X-Secret-Name", secretName)
		req.Header.Set("X-Secret-Version", versionStr)

		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusInternalServerError, rw.Code)
		assert.Contains(t, rw.Body.String(), "ошибка получения секрета")
	})
}
