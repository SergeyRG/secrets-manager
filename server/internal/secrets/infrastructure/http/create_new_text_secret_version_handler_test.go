package http_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	authcontextmanager "github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/auth_context_manager"
	secretDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	handler "github.com/SergeyRG/secrets-manager/server/internal/secrets/infrastructure/http"
	secretUseCases "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases"
	secretsMocks "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases/mocks"
	sharedUseCases "github.com/SergeyRG/secrets-manager/server/internal/shared/use-cases"
)

type StubTxManagerTextVersionHandler struct{}

func (t StubTxManagerTextVersionHandler) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (t StubTxManagerTextVersionHandler) GetExecutor(ctx context.Context) sharedUseCases.QueryExecutor {
	return nil
}

func TestCreateNewTextSecretVersionHandler_Handle(t *testing.T) {
	testUserID := domain.UserID("user-888")
	secretName := "my_text_secret"
	secretDataPayload := "updated-confidential-text-data-payload"

	validJSONBody := `{
		"secret_name": "` + secretName + `",
		"data": "` + secretDataPayload + `"
	}`

	t.Run("Успешное создание новой версии текстового секрета", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSecretMetaRepo := secretsMocks.NewMockSecretMetadataRepository(ctrl)
		mockSecretDataSaver := secretsMocks.NewMockSecretDataSaver(ctrl)

		mockSecretMetaRepo.EXPECT().
			GetUserSecretMetadataByName(gomock.Any(), testUserID, secretName, gomock.Any()).
			Return(secretDomain.SecretsMetadata{
				SecretID:   "secret-uuid-888",
				SecretName: secretName,
				SecretType: secretDomain.SecretTypeFreeText, // Важно: тип FreeText для поиска в мапе сейверов
				Version:    1,
			}, nil).
			Times(1)

		mockSecretMetaRepo.EXPECT().
			AddSecretMetadata(gomock.Any(), gomock.Any()).
			Return(nil).
			Times(1)

		mockSecretDataSaver.EXPECT().
			SaveSecretData(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil).
			Times(1)

		saversMap := map[secretDomain.SecretType]secretUseCases.SecretDataSaver{
			secretDomain.SecretTypeFreeText: mockSecretDataSaver,
		}

		addSecretVersionUC := secretUseCases.NewAddSecretVersionUseCase(
			mockSecretMetaRepo,
			saversMap,
			StubTxManagerTextVersionHandler{},
			zap.NewNop(),
		)
		h := handler.NewCreateNewTextSecretVersionHandler(addSecretVersionUC)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets/text/version", bytes.NewBufferString(validJSONBody))
		req.Header.Set("content-type", "application/json")

		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)

		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusCreated, rw.Code)
	})

	t.Run("Ошибка: Запрос не авторизован (нет UserID в контексте)", func(t *testing.T) {
		h := handler.NewCreateNewTextSecretVersionHandler(nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets/text/version", bytes.NewBufferString(validJSONBody))
		req.Header.Set("content-type", "application/json")
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusUnauthorized, rw.Code)
	})

	t.Run("Ошибка: Невалидный Content-Type", func(t *testing.T) {
		h := handler.NewCreateNewTextSecretVersionHandler(nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets/text/version", bytes.NewBufferString(validJSONBody))
		req.Header.Set("content-type", "multipart/form-data")

		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})

	t.Run("Ошибка: Нарушение структуры JSON (ошибка парсинга)", func(t *testing.T) {
		h := handler.NewCreateNewTextSecretVersionHandler(nil)

		badJSON := `{"secret_name": "name", "data": ` // Сломанный JSON тело
		req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets/text/version", bytes.NewBufferString(badJSON))
		req.Header.Set("content-type", "application/json")

		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
		assert.Contains(t, rw.Body.String(), "ошибка парсинга тела запроса")
	})

	t.Run("Ошибка: Сбой на уровне бизнес-логики UseCase при создании версии", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSecretMetaRepo := secretsMocks.NewMockSecretMetadataRepository(ctrl)
		mockSecretDataSaver := secretsMocks.NewMockSecretDataSaver(ctrl)

		mockSecretMetaRepo.EXPECT().
			GetUserSecretMetadataByName(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(secretDomain.SecretsMetadata{}, errors.New("secret not found in db")).
			Times(1)

		saversMap := map[secretDomain.SecretType]secretUseCases.SecretDataSaver{
			secretDomain.SecretTypeFreeText: mockSecretDataSaver,
		}

		addSecretVersionUC := secretUseCases.NewAddSecretVersionUseCase(
			mockSecretMetaRepo,
			saversMap,
			StubTxManagerTextVersionHandler{},
			zap.NewNop(),
		)
		h := handler.NewCreateNewTextSecretVersionHandler(addSecretVersionUC)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets/text/version", bytes.NewBufferString(validJSONBody))
		req.Header.Set("content-type", "application/json")
		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusInternalServerError, rw.Code)
		assert.Contains(t, rw.Body.String(), "ошибка создания секрета")
	})
}
