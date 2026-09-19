package authclient

import (
	"context"
	"fmt"
	"net/http"

	"github.com/SergeyRG/secrets-manager/client/internal/app/auth/usecases"
	sharedUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/shared/usecases"

	"resty.dev/v3"
)

type restyAuthReqDTO struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type RestyPasswdAuthClient struct {
	client   *resty.Client
	provider PasswdCredsProvider
}

func NewRestyPasswdAuthClient(client *resty.Client, provider PasswdCredsProvider) *RestyPasswdAuthClient {
	return &RestyPasswdAuthClient{client: client, provider: provider}
}

func (a *RestyPasswdAuthClient) Authenticate(ctx context.Context, storage usecases.TokenStorage) error {

	creds, err := a.provider.GetCreds(ctx)

	body := restyAuthReqDTO{
		Login:    creds.Login,
		Password: creds.Passwd,
	}
	resp, err := a.client.NewRequest().
		SetHeader("content-type", "application/json").
		SetBody(body).
		Post("api/user/login")

	if resp.StatusCode() == 401 {
		return usecases.ErrAuthenticationFailed
	}
	if resp.StatusCode() != 200 {
		return sharedUsecases.ErrServerSideError
	}
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		return sharedUsecases.ErrServerSideError
	}

	var authTokenCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "auth_token" {
			authTokenCookie = cookie
			break
		}
	}

	if authTokenCookie == nil {
		return sharedUsecases.ErrServerSideError
	}

	if authTokenCookie.Value == "" {
		return sharedUsecases.ErrServerSideError
	}

	err = storage.SaveRaw(ctx, []byte(authTokenCookie.Value))
	if err != nil {
		return fmt.Errorf("ошибка сохранения токена доступа: %w", err)
	}

	return nil
}
