package resty

import (
	"context"
	"encoding/base64"
	"fmt"

	securityUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/security/usecases"
	"resty.dev/v3"
)

type RestyEncryptedKeyRepo struct {
	restiClient         *resty.Client
	SecurityRelativeUrl string
}

func NewRestyEncryptedKeyRepo(
	restiClient *resty.Client,
	SecurityRelativeUrl string,
) *RestyEncryptedKeyRepo {
	return &RestyEncryptedKeyRepo{
		restiClient:         restiClient,
		SecurityRelativeUrl: SecurityRelativeUrl}
}

type EncryptedKeyRepository interface {
	SaveEncryptedKey(ctx context.Context, encKey []byte) error
	GetEncryptedKey(ctx context.Context) (encKey []byte, err error)
}

func (uc *RestyEncryptedKeyRepo) SaveEncryptedKey(ctx context.Context, encKey []byte) error {
	encodedKey := base64.StdEncoding.EncodeToString(encKey)
	resp, err := uc.restiClient.R().
		SetContext(ctx).
		SetHeader("content-type", "text/plain").
		SetBody(encodedKey).
		Post(uc.SecurityRelativeUrl)
	if err != nil {
		return fmt.Errorf("ошибка отправки http запроса: %w", err)
	}
	if resp.IsStatusFailure() {
		return fmt.Errorf("сервер вернул ошибочный статус: %v.", resp.StatusCode())
	}

	return nil
}

func (uc *RestyEncryptedKeyRepo) GetEncryptedKey(ctx context.Context) (encKey []byte, err error) {
	resp, err := uc.restiClient.R().
		SetContext(ctx).
		SetHeader("Accept", "text/plain").
		Get(uc.SecurityRelativeUrl)
	if err != nil {
		return nil, fmt.Errorf("ошибка отправки http запроса: %w", err)
	}
	if resp.IsStatusFailure() {
		if resp.StatusCode() == 404 {
			return nil, securityUsecases.ErrKeyNotExist
		}
		return nil, fmt.Errorf("сервер вернул ошибочный статус: %v.", resp.StatusCode())
	}

	st := resp.String()

	decodedKey, err := base64.StdEncoding.DecodeString(st)
	if err != nil {
		return nil, fmt.Errorf("ошибка декодирования base64: %w", err)
	}
	return decodedKey, nil
}
