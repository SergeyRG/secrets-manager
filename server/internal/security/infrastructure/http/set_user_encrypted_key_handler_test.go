package http_test

import (
	"bytes"
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
	securityMocks "github.com/SergeyRG/secrets-manager/server/internal/security/use-cases/mocks"
)

func TestSetUserEncryptedKeyHandler_Handle(t *testing.T) {
	testUserID := sharedDomain.UserID("user-security-222")
	rawKeyBytes := []byte("new-super-secure-encrypted-key-payload")
	expectedDomainKey := securityDomain.EncryptedKey(rawKeyBytes)

	t.Run("Успешное сохранение зашифрованного ключа пользователя", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockKeyRepo := securityMocks.NewMockEncryptedKeyRepo(ctrl)

		mockKeyRepo.EXPECT().
			SetUserEncryptedKey(gomock.Any(), testUserID, expectedDomainKey).
			Return(nil).
			Times(1)

		h := securityHttp.NewSetUserEncryptedKeyHandler(mockKeyRepo)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/security/key", bytes.NewBuffer(rawKeyBytes))
		req.Header.Set("content-type", "text/plain; charset=utf-8")

		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)

		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusOK, rw.Code)
	})

	t.Run("Ошибка: Запрос не авторизован (нет UserID в контексте)", func(t *testing.T) {
		h := securityHttp.NewSetUserEncryptedKeyHandler(nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/security/key", bytes.NewBuffer(rawKeyBytes))
		req.Header.Set("content-type", "text/plain")
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusUnauthorized, rw.Code)
	})

	t.Run("Ошибка: Невалидный Content-Type (ожидается text/plain)", func(t *testing.T) {
		h := securityHttp.NewSetUserEncryptedKeyHandler(nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/security/key", bytes.NewBuffer(rawKeyBytes))
		req.Header.Set("content-type", "application/json") // Неверный заголовок

		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})

	t.Run("Ошибка: Сбой репозитория при записи данных в БД (500 Internal Error)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockKeyRepo := securityMocks.NewMockEncryptedKeyRepo(ctrl)

		mockKeyRepo.EXPECT().
			SetUserEncryptedKey(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(errors.New("sql: transaction rollbacked due to deadlock")).
			Times(1)

		h := securityHttp.NewSetUserEncryptedKeyHandler(mockKeyRepo)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/security/key", bytes.NewBuffer(rawKeyBytes))
		req.Header.Set("content-type", "text/plain")

		ctxWithUser := authcontextmanager.ContextWithUserID(req.Context(), testUserID)
		req = req.WithContext(ctxWithUser)
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusInternalServerError, rw.Code)
		assert.Contains(t, rw.Body.String(), "ошибка записи ключа в БД")
	})
}
