package http_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	authcontextmanager "github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/auth_context_manager"
	secretDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	handler "github.com/SergeyRG/secrets-manager/server/internal/secrets/infrastructure/http"
	secretUseCases "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases"
	secretsMocks "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases/mocks"
	sharedUseCases "github.com/SergeyRG/secrets-manager/server/internal/shared/use-cases"
)

type StubTxManagerPageHandler struct{}

func (t StubTxManagerPageHandler) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (t StubTxManagerPageHandler) GetExecutor(ctx context.Context) sharedUseCases.QueryExecutor {
	return nil
}

type FlushableResponseWriter struct {
	*httptest.ResponseRecorder
}

func (f *FlushableResponseWriter) Flush() {}

func TestGetUserSecretMetadataPageHandler_Handle(t *testing.T) {
	testUserID := domain.UserID("user-stream-999")

	t.Run("Успешный стриминг страниц метаданных (NDJSON)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSecretMetaRepo := secretsMocks.NewMockSecretMetadataRepository(ctrl)

		mockSecretMetaRepo.EXPECT().
			GetUserSecretsMetadataPage(gomock.Any(), testUserID, 1, 20).
			Return(func(yield func(secretDomain.SecretsMetadata, error) bool) {
				if !yield(secretDomain.SecretsMetadata{SecretName: "passwords", SecretType: secretDomain.SecretTypeAuthData, Version: 1}, nil) {
					return
				}
				if !yield(secretDomain.SecretsMetadata{SecretName: "notes", SecretType: secretDomain.SecretTypeFreeText, Version: 2}, nil) {
					return
				}
			}).
			Times(1)

		getUserSecretMetadataPageUC := secretUseCases.NewGetUserSecretMetadataPageUseCase(
			mockSecretMetaRepo,
			StubTxManagerPageHandler{},
		)
		h := handler.NewGetUserSecretMetadataPageHandler(getUserSecretMetadataPageUC)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets?page=1&per_page=20", nil)
		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)

		recorder := httptest.NewRecorder()
		rw := &FlushableResponseWriter{ResponseRecorder: recorder}

		h.Handle(rw, req)

		assert.Equal(t, http.StatusOK, rw.Code)
		assert.Equal(t, "application/x-ndjson", rw.Header().Get("Content-Type"))
		assert.Equal(t, "nosniff", rw.Header().Get("X-Content-Type-Options"))

		responseBody := rw.Body.String()
		assert.Contains(t, responseBody, `"secret_name":"passwords"`)
		assert.Contains(t, responseBody, `"secret_type":"SECRET_TYPE_AUTH_DATA"`)
		assert.Contains(t, responseBody, `"secret_name":"notes"`)
		assert.Contains(t, responseBody, `"secret_type":"SECRET_TYPE_FREE_TEXT"`)

		lines := strings.Split(strings.TrimSpace(responseBody), "\n")
		assert.Len(t, lines, 2)
	})

	t.Run("Ошибка: Запрос не авторизован (нет UserID в контексте)", func(t *testing.T) {
		h := handler.NewGetUserSecretMetadataPageHandler(nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets", nil)
		recorder := httptest.NewRecorder()
		rw := &FlushableResponseWriter{ResponseRecorder: recorder}

		h.Handle(rw, req)

		assert.Equal(t, http.StatusUnauthorized, rw.Code)
	})

	t.Run("Ошибка: Падение на первой итерации (ошибка получения данных из БД)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSecretMetaRepo := secretsMocks.NewMockSecretMetadataRepository(ctrl)

		mockSecretMetaRepo.EXPECT().
			GetUserSecretsMetadataPage(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(func(yield func(secretDomain.SecretsMetadata, error) bool) {
				yield(secretDomain.SecretsMetadata{}, errors.New("connection reset"))
			}).
			Times(1)

		getUserSecretMetadataPageUC := secretUseCases.NewGetUserSecretMetadataPageUseCase(
			mockSecretMetaRepo,
			StubTxManagerPageHandler{},
		)
		h := handler.NewGetUserSecretMetadataPageHandler(getUserSecretMetadataPageUC)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets", nil)
		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)

		recorder := httptest.NewRecorder()
		rw := &FlushableResponseWriter{ResponseRecorder: recorder}

		h.Handle(rw, req)

		assert.Equal(t, http.StatusInternalServerError, rw.Code)
		assert.Contains(t, rw.Body.String(), "ошибка получения данных из БД")
	})

	t.Run("Ошибка: Падение в процессе стриминга (после отправки заголовков)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSecretMetaRepo := secretsMocks.NewMockSecretMetadataRepository(ctrl)

		mockSecretMetaRepo.EXPECT().
			GetUserSecretsMetadataPage(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(func(yield func(secretDomain.SecretsMetadata, error) bool) {
				if !yield(secretDomain.SecretsMetadata{SecretName: "valid_one", SecretType: secretDomain.SecretTypeBinary, Version: 1}, nil) {
					return
				}
				yield(secretDomain.SecretsMetadata{}, errors.New("stream database failure"))
			}).
			Times(1)

		getUserSecretMetadataPageUC := secretUseCases.NewGetUserSecretMetadataPageUseCase(
			mockSecretMetaRepo,
			StubTxManagerPageHandler{},
		)
		h := handler.NewGetUserSecretMetadataPageHandler(getUserSecretMetadataPageUC)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/secrets", nil)
		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)

		recorder := httptest.NewRecorder()
		rw := &FlushableResponseWriter{ResponseRecorder: recorder}

		h.Handle(rw, req)

		assert.Equal(t, http.StatusOK, rw.Code)

		responseBody := rw.Body.String()
		assert.Contains(t, responseBody, `"error":"ошибка получения информации об очередном секрете"`)

		lines := strings.Split(strings.TrimSpace(responseBody), "\n")
		assert.Len(t, lines, 2)
	})
}
