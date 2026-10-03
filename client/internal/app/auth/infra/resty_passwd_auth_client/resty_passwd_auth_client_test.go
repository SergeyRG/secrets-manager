package authclient_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authclient "github.com/SergeyRG/secrets-manager/client/internal/app/auth/infra/resty_passwd_auth_client"
	mocks "github.com/SergeyRG/secrets-manager/client/internal/app/auth/infra/resty_passwd_auth_client/mocks"
	"github.com/SergeyRG/secrets-manager/client/internal/app/auth/usecases"
	usecasesMocks "github.com/SergeyRG/secrets-manager/client/internal/app/auth/usecases/mocks"
	sharedUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/shared/usecases"
	"go.uber.org/mock/gomock"
	"resty.dev/v3"
)

func TestRestyPasswdAuthClient_Authenticate(t *testing.T) {
	var serverHandler http.HandlerFunc
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverHandler(w, r)
	}))
	defer ts.Close()

	testCreds := authclient.PasswdCreds{
		Login:  "sergey",
		Passwd: "password123",
	}

	tests := []struct {
		name          string
		mockCreds     func(m *mocks.MockPasswdCredsProvider)
		serverHandler http.HandlerFunc
		mockStorage   func(m *usecasesMocks.MockTokenStorage) // Изменили сигнатуру
		wantLogin     string
		wantErr       error
	}{
		{
			name: "Успешная аутентификация и сохранение токена",
			mockCreds: func(m *mocks.MockPasswdCredsProvider) {
				m.EXPECT().GetCreds(gomock.Any()).Return(testCreds, nil)
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/user/login" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				http.SetCookie(w, &http.Cookie{Name: "auth_token", Value: "jwt-token-xyz"})
				w.WriteHeader(http.StatusOK)
			},
			mockStorage: func(m *usecasesMocks.MockTokenStorage) {
				// Ожидаем успешный вызов сохранения токена
				m.EXPECT().
					SaveRaw(gomock.Any(), []byte("jwt-token-xyz")).
					Return(nil).
					Times(1)
			},
			wantLogin: "sergey",
			wantErr:   nil,
		},
		{
			name: "Ошибка получения учетных данных из провайдера",
			mockCreds: func(m *mocks.MockPasswdCredsProvider) {
				m.EXPECT().GetCreds(gomock.Any()).Return(authclient.PasswdCreds{}, errors.New("CLI error"))
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {},
			mockStorage:   func(m *usecasesMocks.MockTokenStorage) {}, // Вызовов не будет
			wantLogin:     "",
			wantErr:       errors.New("ошибка получения данных для аутентификации"),
		},
		{
			name: "Ошибка сервера: 401 Unauthorized",
			mockCreds: func(m *mocks.MockPasswdCredsProvider) {
				m.EXPECT().GetCreds(gomock.Any()).Return(testCreds, nil)
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			mockStorage: func(m *usecasesMocks.MockTokenStorage) {}, // Вызовов не будет
			wantLogin:   "sergey",
			wantErr:     usecases.ErrAuthenticationFailed,
		},
		{
			name: "Ошибка сервера: 500 Internal Error",
			mockCreds: func(m *mocks.MockPasswdCredsProvider) {
				m.EXPECT().GetCreds(gomock.Any()).Return(testCreds, nil)
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			mockStorage: func(m *usecasesMocks.MockTokenStorage) {},
			wantLogin:   "sergey",
			wantErr:     sharedUsecases.ErrServerSideError,
		},
		{
			name: "Ошибка сервера: Успешный статус, но нет Cookies",
			mockCreds: func(m *mocks.MockPasswdCredsProvider) {
				m.EXPECT().GetCreds(gomock.Any()).Return(testCreds, nil)
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			mockStorage: func(m *usecasesMocks.MockTokenStorage) {},
			wantLogin:   "sergey",
			wantErr:     sharedUsecases.ErrServerSideError,
		},
		{
			name: "Ошибка сохранения токена в хранилище",
			mockCreds: func(m *mocks.MockPasswdCredsProvider) {
				m.EXPECT().GetCreds(gomock.Any()).Return(testCreds, nil)
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				http.SetCookie(w, &http.Cookie{Name: "auth_token", Value: "jwt-token-xyz"})
				w.WriteHeader(http.StatusOK)
			},
			mockStorage: func(m *usecasesMocks.MockTokenStorage) {
				// Имитируем сбой диска/БД при сохранении токена
				m.EXPECT().
					SaveRaw(gomock.Any(), gomock.Any()).
					Return(errors.New("disk full")).
					Times(1)
			},
			wantLogin: "sergey",
			wantErr:   errors.New("ошибка сохранения токена доступа"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProvider := mocks.NewMockPasswdCredsProvider(ctrl)
			tt.mockCreds(mockProvider)

			storage := usecasesMocks.NewMockTokenStorage(ctrl)
			tt.mockStorage(storage) // Конфигурируем мок хранилища для текущего теста

			serverHandler = tt.serverHandler

			restyClient := resty.New().SetBaseURL(ts.URL)
			authCli := authclient.NewRestyPasswdAuthClient(restyClient, mockProvider)

			login, err := authCli.Authenticate(context.Background(), storage)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("ожидалась ошибка %v, но получен nil", tt.wantErr)
				}
				// strings.Contains безопаснее самописного хелпера
				if !errors.Is(err, tt.wantErr) && !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("получена ошибка: %v, ожидалась: %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("не ожидалось ошибки, но получена: %v", err)
			}

			if login != tt.wantLogin {
				t.Errorf("получен login: %q, ожидался: %q", login, tt.wantLogin)
			}
		})
	}
}
