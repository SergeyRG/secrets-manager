package http_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	sharedDomain "github.com/SergeyRG/secrets-manager/internal/shared/domain"
	authcontextmanager "github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/auth_context_manager"
	securityDomain "github.com/SergeyRG/secrets-manager/server/internal/security/domain"
	securityHttp "github.com/SergeyRG/secrets-manager/server/internal/security/infrastructure/http"
	usecases "github.com/SergeyRG/secrets-manager/server/internal/security/use-cases"
	securityMocks "github.com/SergeyRG/secrets-manager/server/internal/security/use-cases/mocks"
)

func TestGetUserEncryptedKeyHandler_Handle(t *testing.T) {
	testUserID := sharedDomain.UserID("user-security-111")
	expectedKey := securityDomain.EncryptedKey("my-secret-encrypted-key-string-data")

	t.Run("Успешное получение зашифрованного ключа пользователя", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockKeyRepo := securityMocks.NewMockEncryptedKeyRepo(ctrl)

		mockKeyRepo.EXPECT().
			GetUserEncryptedKey(gomock.Any(), testUserID).
			Return(expectedKey, nil). // Теперь типы строго совпадают
			Times(1)

		h := securityHttp.NewGetUserEncryptedKeyHandler(mockKeyRepo)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/security/key", nil)

		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)

		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusOK, rw.Code)
		assert.Equal(t, "text/plain", rw.Header().Get("Content-Type"))
		assert.Equal(t, string(expectedKey), rw.Body.String())
	})

	t.Run("Ошибка: Запрос не авторизован (нет UserID в контексте)", func(t *testing.T) {
		h := securityHttp.NewGetUserEncryptedKeyHandler(nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/security/key", nil)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusUnauthorized, rw.Code)
	})

	t.Run("Ошибка: Ключ пользователя не найден в системе (404 Not Found)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockKeyRepo := securityMocks.NewMockEncryptedKeyRepo(ctrl)

		mockKeyRepo.EXPECT().
			GetUserEncryptedKey(gomock.Any(), gomock.Any()).
			Return(securityDomain.EncryptedKey(""), usecases.ErrEncryptionKeyDoesntExists). // ИСПРАВЛЕНО
			Times(1)

		h := securityHttp.NewGetUserEncryptedKeyHandler(mockKeyRepo)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/security/key", nil)
		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusNotFound, rw.Code)
		assert.Contains(t, rw.Body.String(), "ключ пользователя не задан")
	})

	t.Run("Ошибка: Сбой репозитория (500 Internal Server Error)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockKeyRepo := securityMocks.NewMockEncryptedKeyRepo(ctrl)

		mockKeyRepo.EXPECT().
			GetUserEncryptedKey(gomock.Any(), gomock.Any()).
			Return(securityDomain.EncryptedKey(""), errors.New("internal connection error")). // ИСПРАВЛЕНО
			Times(1)

		h := securityHttp.NewGetUserEncryptedKeyHandler(mockKeyRepo)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/security/key", nil)
		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusInternalServerError, rw.Code)
		assert.Contains(t, rw.Body.String(), "ошибка получения ключа")
	})
}
