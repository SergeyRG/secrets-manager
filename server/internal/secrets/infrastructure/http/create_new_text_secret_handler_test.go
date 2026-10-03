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
	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	handler "github.com/SergeyRG/secrets-manager/server/internal/secrets/infrastructure/http"
	secretUseCases "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases"
	secretsMocks "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases/mocks"
	sharedUseCases "github.com/SergeyRG/secrets-manager/server/internal/shared/use-cases"
)

type StubTxManagerTextHandler struct{}

func (t StubTxManagerTextHandler) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (t StubTxManagerTextHandler) GetExecutor(ctx context.Context) sharedUseCases.QueryExecutor {
	return nil
}

func TestCreateNewTextSecretHandler_Handle(t *testing.T) {
	testUserID := domain.UserID("user-999")
	secretName := "my_text_notes"
	secretTypeStr := "SECRET_TYPE_FREE_TEXT"
	secretDataPayload := "this-is-some-raw-confidential-text"

	validJSONBody := `{
		"secret_name": "` + secretName + `",
		"secret_type": "` + secretTypeStr + `",
		"data": "` + secretDataPayload + `"
	}`

	t.Run("Успешное создание текстового секрета", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSecretMetaRepo := secretsMocks.NewMockSecretMetadataRepository(ctrl)
		mockSecretDataSaver := secretsMocks.NewMockSecretDataSaver(ctrl)

		mockSecretMetaRepo.EXPECT().
			AddSecretMetadata(gomock.Any(), gomock.Any()).
			Return(nil).
			Times(1)

		mockSecretDataSaver.EXPECT().
			SaveSecretData(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil).
			Times(1)

		saversMap := map[secretsDomain.SecretType]secretUseCases.SecretDataSaver{
			secretsDomain.SecretTypeFreeText: mockSecretDataSaver,
		}

		createSecretUC := secretUseCases.NewCreateNewSecretUseCase(
			mockSecretMetaRepo,
			saversMap,
			StubTxManagerTextHandler{},
			zap.NewNop(),
		)
		h := handler.NewCreateNewTextSecretHandler(createSecretUC)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets/text", bytes.NewBufferString(validJSONBody))
		req.Header.Set("content-type", "application/json; charset=utf-8")

		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)

		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusCreated, rw.Code)
	})

	t.Run("Ошибка: Запрос не авторизован (нет UserID в контексте)", func(t *testing.T) {
		h := handler.NewCreateNewTextSecretHandler(nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets/text", bytes.NewBufferString(validJSONBody))
		req.Header.Set("content-type", "application/json")
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusUnauthorized, rw.Code)
	})

	t.Run("Ошибка: Невалидный Content-Type в заголовках", func(t *testing.T) {
		h := handler.NewCreateNewTextSecretHandler(nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets/text", bytes.NewBufferString(validJSONBody))
		req.Header.Set("content-type", "text/plain") // Передаем некорректный заголовок

		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})

	t.Run("Ошибка: Битая структура JSON тела запроса (ошибка парсинга)", func(t *testing.T) {
		h := handler.NewCreateNewTextSecretHandler(nil)

		invalidJSON := `{"secret_name": "valid_name", "secret_type": "SECRET_TYPE_FREE_TEXT", "data": ` // Обрезанный JSON
		req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets/text", bytes.NewBufferString(invalidJSON))
		req.Header.Set("content-type", "application/json")

		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
		assert.Contains(t, rw.Body.String(), "ошибка парсинга тела запроса")
	})

	t.Run("Ошибка: Сбой на уровне бизнес-логики UseCase (например, падение БД)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSecretMetaRepo := secretsMocks.NewMockSecretMetadataRepository(ctrl)
		mockSecretDataSaver := secretsMocks.NewMockSecretDataSaver(ctrl)

		mockSecretMetaRepo.EXPECT().
			AddSecretMetadata(gomock.Any(), gomock.Any()).
			Return(errors.New("internal database query execution error")).
			Times(1)

		saversMap := map[secretsDomain.SecretType]secretUseCases.SecretDataSaver{
			secretsDomain.SecretTypeFreeText: mockSecretDataSaver,
		}

		createSecretUC := secretUseCases.NewCreateNewSecretUseCase(
			mockSecretMetaRepo,
			saversMap,
			StubTxManagerTextHandler{},
			zap.NewNop(),
		)
		h := handler.NewCreateNewTextSecretHandler(createSecretUC)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets/text", bytes.NewBufferString(validJSONBody))
		req.Header.Set("content-type", "application/json")
		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusInternalServerError, rw.Code)
		assert.Contains(t, rw.Body.String(), "ошибка создания секрета")
	})
}
