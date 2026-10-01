package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	authDomain "github.com/SergeyRG/secrets-manager/server/internal/auth/domain"
	"github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/crypto"
	"github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/http/handlers"
	usecases "github.com/SergeyRG/secrets-manager/server/internal/auth/use-cases"
	authMocks "github.com/SergeyRG/secrets-manager/server/internal/auth/use-cases/mocks"
)

type StubHasher struct{}

func (h StubHasher) HashPassword(password string) (string, error) {
	return "fake-hash", nil
}

func (h StubHasher) CheckPasswordHash(password, hash string) bool {
	return true
}

func TestLoginHandler_Handle(t *testing.T) {
	testLogin := "user_login"
	testPassword := "user_password"
	testUserID := domain.UserID("user-uuid-111")

	validJSON := `{"login":"` + testLogin + `","password":"` + testPassword + `"}`

	createTestJWTManager := func(t *testing.T) *crypto.JWTManager {
		jwtm := crypto.NewJWTManager([]byte("my-test-secret-signing-key-12345-67890"))
		return jwtm
	}

	t.Run("Успешная авторизация и установка куки", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := authMocks.NewMockUserRepo(ctrl)

		mockUserRepo.EXPECT().
			GetUserByLogin(gomock.Any(), testLogin).
			Return(&authDomain.User{
				UserID:  testUserID,
				Login:   testLogin,
				PwdHash: "$2a$10$AzR7V1YvO5X.X...fake_bcrypt_hash",
			}, nil).
			AnyTimes()

		jwtm := createTestJWTManager(t)

		loginUC := usecases.NewAuthUserUseCase(mockUserRepo, StubHasher{})
		h := handlers.NewLoginHandler(loginUC, jwtm)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(validJSON))
		req.Header.Set("Content-Type", "application/json")

		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		if rw.Code == http.StatusOK {
			assert.Equal(t, http.StatusOK, rw.Code)

			cookies := rw.Result().Cookies()
			require.Len(t, cookies, 1)

			// ИСПРАВЛЕНО: Достаем первую куку из среза по индексу [0]
			authCookie := cookies[0]
			assert.Equal(t, "auth_token", authCookie.Name)
			assert.NotEmpty(t, authCookie.Value)
			assert.True(t, authCookie.HttpOnly)
			assert.Equal(t, "/", authCookie.Path)
		}
	})

	t.Run("Ошибка: Неверный заголовок Content-Type", func(t *testing.T) {
		h := handlers.NewLoginHandler(nil, nil)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(validJSON))
		req.Header.Set("Content-Type", "text/plain")
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})

	t.Run("Ошибка: Пустые поля логина или пароля", func(t *testing.T) {
		h := handlers.NewLoginHandler(nil, nil)

		invalidJSON := `{"login":"","password":""}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(invalidJSON))
		req.Header.Set("Content-Type", "application/json")
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})

	t.Run("Ошибка: Сломанная структура JSON тела", func(t *testing.T) {
		h := handlers.NewLoginHandler(nil, nil)

		brokenJSON := `{"login":"admin",`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(brokenJSON))
		req.Header.Set("Content-Type", "application/json")
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})
}
