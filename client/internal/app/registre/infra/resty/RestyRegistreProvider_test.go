package resty

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SergeyRG/secrets-manager/client/internal/app/registre/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"resty.dev/v3"
)

func TestRestyRegistreProvider_Registre(t *testing.T) {
	testUser := domain.User{
		Login:  "test_user",
		Passwd: "secure_password",
	}

	t.Run("Успешная регистрация", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "application/json", r.Header.Get("content-type"))

			var body registreReq
			err := json.NewDecoder(r.Body).Decode(&body)
			require.NoError(t, err)
			assert.Equal(t, testUser.Login, body.Login)
			assert.Equal(t, testUser.Passwd, body.Password)

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		cl := resty.New()
		defer cl.Close()

		provider := NewRestyRegistreProvider(cl, server.URL)

		err := provider.Registre(context.Background(), testUser)

		assert.NoError(t, err)
	})

	t.Run("Ошибка сервера (код 400 или 500)", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest) // 400 Bad Request
		}))
		defer server.Close()

		cl := resty.New()
		defer cl.Close()

		provider := NewRestyRegistreProvider(cl, server.URL)

		err := provider.Registre(context.Background(), testUser)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "сервер вернул ошибочный код:400")
	})

	t.Run("Сетевая ошибка / Недоступность сервера", func(t *testing.T) {
		cl := resty.New()
		defer cl.Close()

		provider := NewRestyRegistreProvider(cl, "http://invalid-localhost-url.local")

		err := provider.Registre(context.Background(), testUser)

		assert.Error(t, err)
	})
}
