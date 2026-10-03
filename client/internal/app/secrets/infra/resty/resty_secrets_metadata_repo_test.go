package resty_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	secretResty "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/infra/resty"
	secretsUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/usecases"
	"resty.dev/v3"
)

func TestRestySecretsRepo_GetUserSecretMetadataByName(t *testing.T) {
	tests := []struct {
		name          string
		secretName    string
		version       int
		serverHandler http.HandlerFunc
		wantMeta      secretsDomain.SecretsMetadata
		wantErr       bool
	}{
		{
			name:       "Успешное получение метаданных по имени",
			secretName: "my_key",
			version:    1,
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Secret-Name") != "my_key" || r.Header.Get("X-Secret-Version") != "1" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.Header().Set("content-type", "application/json")
				w.WriteHeader(http.StatusOK)

				resp := map[string]interface{}{
					"secret_name": "my_key",
					"version":     1,
					"secret_type": "FreeText",
				}
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantMeta: secretsDomain.SecretsMetadata{
				SecretName: "my_key",
				SecretType: secretsDomain.SecretTypeFromString("FreeText"),
				Version:    1,
			},
			wantErr: false,
		},
		{
			name:       "Ошибка: Сервер вернул 404",
			secretName: "missing_key",
			version:    1,
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantMeta: secretsDomain.SecretsMetadata{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			restyClient := resty.New().SetBaseURL(server.URL)
			repo := secretResty.NewRestySecretsRepo(restyClient, "/api/meta", "/api/list")

			meta, err := repo.GetUserSecretMetadataByName(context.Background(), tt.secretName, tt.version)

			if (err != nil) != tt.wantErr {
				t.Fatalf("GetUserSecretMetadataByName() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && !reflect.DeepEqual(meta, tt.wantMeta) {
				t.Errorf("GetUserSecretMetadataByName() = %+v, want %+v", meta, tt.wantMeta)
			}
		})
	}
}

func TestRestySecretsRepo_GetUserSecretsMetadataPage(t *testing.T) {
	tests := []struct {
		name          string
		page          int
		perPage       int
		serverHandler http.HandlerFunc
		wantList      []secretsDomain.SecretsMetadata
		errCheck      func(error) bool
		wantErr       bool
	}{
		{
			name:    "Успешное получение страницы (NDJSON парсинг)",
			page:    1,
			perPage: 2,
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("page") != "1" || r.URL.Query().Get("per_page") != "2" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.Header().Set("content-type", "application/x-ndjson")
				w.WriteHeader(http.StatusOK)

				// Имитируем NDJSON (разделенные переносом строки json-объекты)
				_, _ = w.Write([]byte(`{"secret_name":"key_1","version":1,"secret_type":"FreeText"}` + "\n"))
				_, _ = w.Write([]byte(`{"secret_name":"key_2","version":3,"secret_type":"Binary"}` + "\n"))
			},
			wantList: []secretsDomain.SecretsMetadata{
				{SecretName: "key_1", SecretType: secretsDomain.SecretTypeFromString("FreeText"), Version: 1},
				{SecretName: "key_2", SecretType: secretsDomain.SecretTypeFromString("Binary"), Version: 3},
			},
			wantErr: false,
		},
		{
			name:    "Успешно: пропуск пустых строк в NDJSON",
			page:    1,
			perPage: 2,
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("\n\n" + `{"secret_name":"key_1","version":1,"secret_type":"FreeText"}` + "\n\n"))
			},
			wantList: []secretsDomain.SecretsMetadata{
				{SecretName: "key_1", SecretType: secretsDomain.SecretTypeFromString("FreeText"), Version: 1},
			},
			wantErr: false,
		},
		{
			name:    "Ошибка: Сервер недоступен (ErrServerUnavailable маркер)",
			page:    1,
			perPage: 2,
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				// Ломаем соединение
				hj, ok := w.(http.Hijacker)
				if ok {
					conn, _, _ := hj.Hijack()
					_ = conn.Close()
				}
			},
			wantList: nil,
			errCheck: func(err error) bool {
				return errors.Is(err, secretsUsecases.ErrServerUnavailable)
			},
			wantErr: true,
		},
		{
			name:    "Ошибка: Битный/битый JSON внутри NDJSON потока",
			page:    1,
			perPage: 2,
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"secret_name": invalid-json-here` + "\n"))
			},
			wantList: nil,
			errCheck: func(err error) bool {
				return strings.Contains(err.Error(), "ошибка парсинга строки ndjson")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			restyClient := resty.New().SetBaseURL(server.URL)
			repo := secretResty.NewRestySecretsRepo(restyClient, "/api/meta", "/api/list")

			list, err := repo.GetUserSecretsMetadataPage(context.Background(), tt.page, tt.perPage)

			if (err != nil) != tt.wantErr {
				t.Fatalf("GetUserSecretsMetadataPage() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.errCheck != nil {
				if !tt.errCheck(err) {
					t.Errorf("Получена неверная структура ошибки: %v", err)
				}
			}

			if !tt.wantErr && !reflect.DeepEqual(list, tt.wantList) {
				t.Errorf("GetUserSecretsMetadataPage() = %+v, want %+v", list, tt.wantList)
			}
		})
	}
}

func TestRestySecretsRepo_AddSecretMetadata(t *testing.T) {
	// Метод AddSecretMetadata содержит заглушку return nil. Покрываем его для 100% покрытия пакета.
	client := resty.New()
	repo := secretResty.NewRestySecretsRepo(client, "", "")
	err := repo.AddSecretMetadata(context.Background(), secretsDomain.SecretsMetadata{})
	if err != nil {
		t.Errorf("Ожидался nil при вызове заглушки, получено: %v", err)
	}
}
