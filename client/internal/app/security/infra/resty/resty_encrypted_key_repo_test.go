package resty_test

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	// Измените путь на ваш реальный путь пакета
	"github.com/SergeyRG/secrets-manager/client/internal/app/security/infra/resty"
	securityUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/security/usecases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	restyV3 "resty.dev/v3"
)

func TestRestyEncryptedKeyRepo(t *testing.T) {
	rawKey := []byte("my-super-secret-key-bytes")
	base64Key := base64.StdEncoding.EncodeToString(rawKey)

	t.Run("SaveEncryptedKey - Успех", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "text/plain", r.Header.Get("content-type"))

			buf := make([]byte, len(base64Key)+10)
			n, _ := r.Body.Read(buf)
			assert.Equal(t, base64Key, string(buf[:n]))

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		cl := restyV3.New()
		defer cl.Close()

		repo := resty.NewRestyEncryptedKeyRepo(cl, server.URL)
		err := repo.SaveEncryptedKey(context.Background(), rawKey)

		assert.NoError(t, err)
	})

	t.Run("SaveEncryptedKey - Ошибка сервера", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError) // 500
		}))
		defer server.Close()

		cl := restyV3.New()
		defer cl.Close()

		repo := resty.NewRestyEncryptedKeyRepo(cl, server.URL)
		err := repo.SaveEncryptedKey(context.Background(), rawKey)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "сервер вернул ошибочный статус: 500")
	})

	t.Run("GetEncryptedKey - Успех", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "text/plain", r.Header.Get("Accept"))

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(base64Key))
		}))
		defer server.Close()

		cl := restyV3.New()
		defer cl.Close()

		repo := resty.NewRestyEncryptedKeyRepo(cl, server.URL)
		key, err := repo.GetEncryptedKey(context.Background())

		require.NoError(t, err)
		assert.Equal(t, rawKey, key)
	})

	t.Run("GetEncryptedKey - Ключ не найден (404)", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound) // 404
		}))
		defer server.Close()

		cl := restyV3.New()
		defer cl.Close()

		repo := resty.NewRestyEncryptedKeyRepo(cl, server.URL)
		key, err := repo.GetEncryptedKey(context.Background())

		assert.Nil(t, key)
		assert.ErrorIs(t, err, securityUsecases.ErrKeyNotExist)
	})

	t.Run("GetEncryptedKey - Ошибка декодирования base64", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not-a-valid-base64-!!!")) // Невалидный base64
		}))
		defer server.Close()

		cl := restyV3.New()
		defer cl.Close()

		repo := resty.NewRestyEncryptedKeyRepo(cl, server.URL)
		key, err := repo.GetEncryptedKey(context.Background())

		assert.Nil(t, key)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка декодирования base64")
	})

	t.Run("Сетевая ошибка / Сервер недоступен", func(t *testing.T) {
		cl := restyV3.New()
		defer cl.Close()

		repo := resty.NewRestyEncryptedKeyRepo(cl, "http://invalid-localhost-domain.local")

		key, err := repo.GetEncryptedKey(context.Background())

		assert.Nil(t, key)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка отправки http запроса")
	})
}
