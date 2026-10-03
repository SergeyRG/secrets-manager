package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	authcontextmanager "github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/auth_context_manager"
	secretDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	handler "github.com/SergeyRG/secrets-manager/server/internal/secrets/infrastructure/http"
	secretUseCases "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases"
	secretsMocks "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases/mocks"
	sharedUseCases "github.com/SergeyRG/secrets-manager/server/internal/shared/use-cases"
)

type StubTxManagerGetTextVersionHandler struct{}

func (t StubTxManagerGetTextVersionHandler) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (t StubTxManagerGetTextVersionHandler) GetExecutor(ctx context.Context) sharedUseCases.QueryExecutor {
	return nil
}

func TestGetTextSecretVersionHandler_Handle(t *testing.T) {
	testUserID := domain.UserID("user-text-456")
	secretName := "my_notes_secret"
	versionStr := "2"
	versionInt := 2
	secretDataPayload := "confidential-text-notes-body"

	t.Run("Успешное получение текстового секрета и возврат JSON", func(t *testing.T) {
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
				SecretType: secretDomain.SecretTypeFreeText,
				Version:    int64(versionInt),
				VersionID:  "version-uuid-2",
			}, nil).
			Times(1)

		mockStream := io.NopCloser(strings.NewReader(secretDataPayload))
		mockSecretReceiver.EXPECT().
			ReceiveSecretData(gomock.Any(), gomock.Any()).
			Return(mockStream, nil).
			Times(1)

		receiversMap := map[secretDomain.SecretType]secretUseCases.SecretDataReceiver{
			secretDomain.SecretTypeFreeText: mockSecretReceiver,
		}

		getSecretVersionUC := secretUseCases.NewGetSecretVersionUseCase(
			mockSecretMetaRepo,
			receiversMap,
			StubTxManagerGetTextVersionHandler{},
			zap.NewNop(),
		)
		h := handler.NewGetTextSecretVersionHandler(getSecretVersionUC)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/text/version", nil)
		req.Header.Set("X-Secret-Name", secretName)
		req.Header.Set("X-Secret-Version", versionStr)

		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)

		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusOK, rw.Code)
		assert.Equal(t, "application/json", rw.Header().Get("Content-Type"))

		var jsonResponse map[string]any
		err := json.Unmarshal(rw.Body.Bytes(), &jsonResponse)
		require.NoError(t, err)

		assert.Equal(t, secretName, jsonResponse["secret_name"])
		assert.Equal(t, float64(versionInt), jsonResponse["version"])
		assert.Equal(t, secretDataPayload, jsonResponse["data"])
	})

	t.Run("Ошибка: Запрос не авторизован (нет UserID в контексте)", func(t *testing.T) {
		h := handler.NewGetTextSecretVersionHandler(nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/text/version", nil)
		req.Header.Set("X-Secret-Name", secretName)
		req.Header.Set("X-Secret-Version", versionStr)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusUnauthorized, rw.Code)
	})

	t.Run("Ошибка: Не заданы необходимые заголовки (X-Secret-Version)", func(t *testing.T) {
		h := handler.NewGetTextSecretVersionHandler(nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/text/version", nil)
		req.Header.Set("X-Secret-Name", secretName)

		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
		assert.Contains(t, rw.Body.String(), "не заданы необходимые заголовки")
	})

	t.Run("Ошибка: Невалидный тип версии в заголовке", func(t *testing.T) {
		h := handler.NewGetTextSecretVersionHandler(nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/text/version", nil)
		req.Header.Set("X-Secret-Name", secretName)
		req.Header.Set("X-Secret-Version", "abc")

		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
		assert.Contains(t, rw.Body.String(), "версия должна быть целым числом")
	})

	t.Run("Ошибка: Сбой UseCase (например, репозиторий вернул ошибку)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSecretMetaRepo := secretsMocks.NewMockSecretMetadataRepository(ctrl)

		mockSecretMetaRepo.EXPECT().
			GetUserSecretMetadataByName(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(secretDomain.SecretsMetadata{}, errors.New("db disconnect error")).
			Times(1)

		getSecretVersionUC := secretUseCases.NewGetSecretVersionUseCase(
			mockSecretMetaRepo,
			nil,
			StubTxManagerGetTextVersionHandler{},
			zap.NewNop(),
		)
		h := handler.NewGetTextSecretVersionHandler(getSecretVersionUC)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/text/version", nil)
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
