package resty

import (
	"context"
	"fmt"

	"github.com/SergeyRG/secrets-manager/client/internal/app/registre/domain"

	"resty.dev/v3"
)

type registreReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type RestyRegistreProvider struct {
	cl                  *resty.Client
	registreRelativeURL string
}

func NewRestyRegistreProvider(cl *resty.Client, url string) *RestyRegistreProvider {
	return &RestyRegistreProvider{cl: cl, registreRelativeURL: url}
}

func (r *RestyRegistreProvider) Registre(ctx context.Context, u domain.User) error {
	req := registreReq{
		Login:    u.Login,
		Password: u.Passwd,
	}
	resp, err := r.cl.R().SetContext(ctx).
		SetHeader("content-type", "application/json").
		SetBody(req).
		Post(r.registreRelativeURL)

	if err != nil {
		return err
	}

	if resp.IsStatusFailure() {
		return fmt.Errorf("сервер вернул ошибочный код:%v", resp.StatusCode())
	}

	return nil
}
