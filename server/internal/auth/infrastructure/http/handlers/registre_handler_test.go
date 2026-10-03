package handlers_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/crypto"
	"github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/http/handlers"
	usecases "github.com/SergeyRG/secrets-manager/server/internal/auth/use-cases"
	authMocks "github.com/SergeyRG/secrets-manager/server/internal/auth/use-cases/mocks"
)

type StubRegisterHasher struct{}

func (h StubRegisterHasher) HashPassword(password string) (string, error) {
	return "fake-hashed-password", nil
}

func (h StubRegisterHasher) CheckPasswordHash(password, hash string) bool {
	return true
}

func TestRegistreHandler_Handle(t *testing.T) {
	testLogin := "new_user"
	testPassword := "secure_pass"

	validJSON := `{"login":"` + testLogin + `","password":"` + testPassword + `"}`

	createTestJWTManager := func() *crypto.JWTManager {
		return crypto.NewJWTManager([]byte("my-test-secret-signing-key-12345-67890"))
	}

	t.Run("Успешная регистрация, установка куки и статус 200 OK", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := authMocks.NewMockUserRepo(ctrl)

		// Ожидаем успешную проверку отсутствия пользователя и его последующее добавление
		mockUserRepo.EXPECT().
			GetUserByLogin(gomock.Any(), testLogin).
			Return(nil, errors.New("sql: no rows in result set")). // Пользователь не найден, путь свободен
			AnyTimes()

		mockUserRepo.EXPECT().
			AddUser(gomock.Any(), gomock.Any()).
			Return(nil).
			Times(1)

		jwtm := createTestJWTManager()

		// Собираем UseCase (сопоставьте аргументы со своим конструктором, если передаются другие сущности)
		registerUC := usecases.NewRegisterUserUseCase(mockUserRepo, StubRegisterHasher{})
		h := handlers.NewRegistreHandler(registerUC, jwtm)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(validJSON))
		req.Header.Set("Content-Type", "application/json")

		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusOK, rw.Code)

		// Извлекаем куки из среза по индексу
		cookies := rw.Result().Cookies()
		require.Len(t, cookies, 1)
		authCookie := cookies[0]

		assert.Equal(t, "auth_token", authCookie.Name)
		assert.NotEmpty(t, authCookie.Value)
		assert.True(t, authCookie.HttpOnly)
		assert.Equal(t, "/", authCookie.Path)
	})

	t.Run("Ошибка: Логин уже занят (Статус 409 Conflict)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := authMocks.NewMockUserRepo(ctrl)

		// Имитируем доменную ошибку ErrLoginBusy, которая выбрасывается UseCase-ом
		// (Либо на уровне репозитория AddUser возвращает ошибку дубликата)
		mockUserRepo.EXPECT().
			AddUser(gomock.Any(), gomock.Any()).
			Return(usecases.ErrLoginBusy).
			AnyTimes()

		mockUserRepo.EXPECT().
			GetUserByLogin(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("no rows")).
			AnyTimes()

		jwtm := createTestJWTManager()
		registerUC := usecases.NewRegisterUserUseCase(mockUserRepo, StubRegisterHasher{})
		h := handlers.NewRegistreHandler(registerUC, jwtm)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(validJSON))
		req.Header.Set("Content-Type", "application/json")
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		// Хендлер должен отдать статус 409 Conflict
		assert.Equal(t, http.StatusConflict, rw.Code)
	})

	t.Run("Ошибка: Пустые обязательные поля", func(t *testing.T) {
		h := handlers.NewRegistreHandler(nil, nil)

		invalidJSON := `{"login":"","password":""}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(invalidJSON))
		req.Header.Set("Content-Type", "application/json")
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})

	t.Run("Ошибка: Невалидный Content-Type", func(t *testing.T) {
		h := handlers.NewRegistreHandler(nil, nil)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(validJSON))
		req.Header.Set("Content-Type", "text/plain")
		rw := httptest.NewRecorder()

		h.Handle(rw, req)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})
}
