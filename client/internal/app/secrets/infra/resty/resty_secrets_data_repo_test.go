package resty_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	secretResty "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/infra/resty"
	"resty.dev/v3"
)

func TestRestySecretsDataRepo_AddSecretData(t *testing.T) {
	tests := []struct {
		name           string
		secretMetadata secretsDomain.SecretsMetadata
		inputData      string
		serverHandler  http.HandlerFunc
		wantErr        bool
	}{
		{
			name: "Успешное добавление текстового секрета (JSON)",
			secretMetadata: secretsDomain.SecretsMetadata{
				SecretName: "my_text_secret",
				SecretType: secretsDomain.SecretTypeFreeText,
			},
			inputData: "text_secret_payload",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/api/text" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if !strings.Contains(r.Header.Get("content-type"), "application/json") {
					w.WriteHeader(http.StatusUnsupportedMediaType)
					return
				}

				bodyBytes, err := io.ReadAll(r.Body)
				if err != nil || len(bodyBytes) == 0 {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name: "Успешное добавление бинарного секрета (Multipart Form)",
			secretMetadata: secretsDomain.SecretsMetadata{
				SecretName: "my_blob_secret",
				SecretType: secretsDomain.SecretTypeBinary,
			},
			inputData: "binary_bytes_content",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/api/blob" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				err := r.ParseMultipartForm(10 << 20)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				if r.FormValue("secretName") != "my_blob_secret" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				file, _, err := r.FormFile("file")
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				defer file.Close()

				fileContent, _ := io.ReadAll(file)
				if string(fileContent) != "binary_bytes_content" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name: "Ошибка сервера (500 Internal Server Error)",
			secretMetadata: secretsDomain.SecretsMetadata{
				SecretName: "broken_secret",
				SecretType: secretsDomain.SecretTypeFreeText,
			},
			inputData: "payload",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			restyClient := resty.New().SetBaseURL(server.URL)
			repo := secretResty.NewRestySecretsDataRepo(restyClient, "/api/text", "/api/blob")

			stream := io.NopCloser(bytes.NewBufferString(tt.inputData))
			err := repo.AddSecretData(context.Background(), tt.secretMetadata, stream)

			if (err != nil) != tt.wantErr {
				t.Errorf("AddSecretData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRestySecretsDataRepo_GetSecretData(t *testing.T) {
	secretContent := "retrieved_secret_data"

	tests := []struct {
		name           string
		secretMetadata secretsDomain.SecretsMetadata
		serverHandler  http.HandlerFunc
		wantData       string
		wantErr        bool
	}{
		{
			name: "Успешное получение текстового секрета через JSON",
			secretMetadata: secretsDomain.SecretsMetadata{
				SecretName: "get_text",
				SecretType: secretsDomain.SecretTypeFreeText,
				Version:    2,
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Secret-Name") != "get_text" || r.Header.Get("X-Secret-Version") != "2" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				w.Header().Set("content-type", "application/json")
				w.WriteHeader(http.StatusOK)

				respStruct := map[string]interface{}{
					"secret_name": "get_text",
					"version":     2,
					"data":        []byte(secretContent),
				}
				_ = json.NewEncoder(w).Encode(respStruct)
			},
			wantData: secretContent,
			wantErr:  false,
		},
		{
			name: "Успешное получение бинарного секрета через сырой поток",
			secretMetadata: secretsDomain.SecretsMetadata{
				SecretName: "get_blob",
				SecretType: secretsDomain.SecretTypeBinary,
				Version:    5,
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Secret-Name") != "get_blob" || r.Header.Get("X-Secret-Version") != "5" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(secretContent))
			},
			wantData: secretContent,
			wantErr:  false,
		},
		{
			name: "Ошибка: сервер вернул 404",
			secretMetadata: secretsDomain.SecretsMetadata{
				SecretName: "missing",
				SecretType: secretsDomain.SecretTypeFreeText,
				Version:    1,
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantData: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			restyClient := resty.New().SetBaseURL(server.URL)
			repo := secretResty.NewRestySecretsDataRepo(restyClient, "/api/text", "/api/blob")

			stream, err := repo.GetSecretData(context.Background(), tt.secretMetadata)

			if (err != nil) != tt.wantErr {
				t.Fatalf("GetSecretData() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			defer stream.Close()
			resBytes, err := io.ReadAll(stream)
			if err != nil {
				t.Fatalf("ошибка вычитки результирующего потока: %v", err)
			}

			if string(resBytes) != tt.wantData {
				t.Errorf("GetSecretData() получено = %q, ожидали = %q", string(resBytes), tt.wantData)
			}
		})
	}
}
