package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	authcontextmanager "github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/auth_context_manager"
	secretDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	handler "github.com/SergeyRG/secrets-manager/server/internal/secrets/infrastructure/http"
	secretUseCases "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases"
	secretsMocks "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases/mocks"
	sharedUseCases "github.com/SergeyRG/secrets-manager/server/internal/shared/use-cases"
)

type StubTxManagerMetadataHandler struct{}

func (t StubTxManagerMetadataHandler) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (t StubTxManagerMetadataHandler) GetExecutor(ctx context.Context) sharedUseCases.QueryExecutor {
	return nil
}

func TestGetUserSecretMetadataHandler_Handle(t *testing.T) {
	testUserID := domain.UserID("user-meta-555")
	secretName := "my_bank_card_secret"
	versionStr := "1"
	versionInt := 1

	t.Run("Успешное получение метаданных секрета и возврат JSON", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSecretMetaRepo := secretsMocks.NewMockSecretMetadataRepository(ctrl)

		mockSecretMetaRepo.EXPECT().
			GetUserSecretMetadataByName(gomock.Any(), testUserID, secretName, versionInt).
			Return(secretDomain.SecretsMetadata{
				UserID:     testUserID,
				SecretID:   "secret-uuid-555",
				SecretName: secretName,
				SecretType: secretDomain.SecretTypeBankCard, // Проверим тип BankCard
				Version:    int64(versionInt),
			}, nil).
			Times(1)

		getUserSecretMetadataUC := secretUseCases.NewGetUserSecretMetadataUseCase(
			mockSecretMetaRepo,
		)
		h := handler.NewGetUserSecretMetadataHandler(getUserSecretMetadataUC)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/metadata", nil)
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
		assert.Equal(t, "SECRET_TYPE_BANK_CARD", jsonResponse["secret_type"]) // Проверка вызова .ToString()
	})

	t.Run("Ошибка: Запрос не авторизован (нет UserID в контексте)", func(t *testing.T) {
		h := handler.NewGetUserSecretMetadataHandler(nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/metadata", nil)
		req.Header.Set("X-Secret-Name", secretName)
		req.Header.Set("X-Secret-Version", versionStr)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusUnauthorized, rw.Code)
	})

	t.Run("Ошибка: Не заданы необходимые заголовки (X-Secret-Name)", func(t *testing.T) {
		h := handler.NewGetUserSecretMetadataHandler(nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/metadata", nil)
		req.Header.Set("X-Secret-Version", versionStr)

		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
		assert.Contains(t, rw.Body.String(), "не заданы необходимые заголовки")
	})

	t.Run("Ошибка: Невалидный формат версии в заголовке", func(t *testing.T) {
		h := handler.NewGetUserSecretMetadataHandler(nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/metadata", nil)
		req.Header.Set("X-Secret-Name", secretName)
		req.Header.Set("X-Secret-Version", "1.2.3") // Строка с точками вместо целого числа

		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
		assert.Contains(t, rw.Body.String(), "версия должна быть целым числом")
	})

	t.Run("Ошибка: Сбой UseCase (например, репозиторий вернул ошибку БД)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSecretMetaRepo := secretsMocks.NewMockSecretMetadataRepository(ctrl)

		mockSecretMetaRepo.EXPECT().
			GetUserSecretMetadataByName(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(secretDomain.SecretsMetadata{}, errors.New("postgres connection timeout")).
			Times(1)

		getUserSecretMetadataUC := secretUseCases.NewGetUserSecretMetadataUseCase(
			mockSecretMetaRepo,
		)
		h := handler.NewGetUserSecretMetadataHandler(getUserSecretMetadataUC)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/metadata", nil)
		req.Header.Set("X-Secret-Name", secretName)
		req.Header.Set("X-Secret-Version", versionStr)

		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusInternalServerError, rw.Code)
		assert.Contains(t, rw.Body.String(), "ошибка получения данных из БД")
	})
}
