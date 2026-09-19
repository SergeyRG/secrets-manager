package resty

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	"resty.dev/v3"
)

type getSecretDataResp struct {
	SecretName string `json:"secret_name"`
	Version    int64  `json:"version"`
	Data       []byte `json:"data"`
}

type createSecretDataReq struct {
	SecretName string `json:"secret_name"`
	SecretType string `json:"secret_type"`
	Data       []byte `json:"data"`
}

type restySecretsDataRepo struct {
	restiClient          *resty.Client
	textDataRelaitiveURL string
	blobDataRelaitiveURL string
}

func NewRestySecretsDataRepo(
	restiClient *resty.Client,
	textDataRelaitiveURL string,
	blobDataRelaitiveURL string,
) *restySecretsDataRepo {
	return &restySecretsDataRepo{
		restiClient:          restiClient,
		textDataRelaitiveURL: textDataRelaitiveURL,
		blobDataRelaitiveURL: blobDataRelaitiveURL,
	}
}

func (r *restySecretsDataRepo) AddSecretData(ctx context.Context, smt secretsDomain.SecretsMetadata, data io.ReadCloser) error {

	switch smt.SecretType {
	case secretsDomain.SecretTypeBinary:

		resp, err := r.restiClient.R().
			SetContext(ctx).
			SetFormData(map[string]string{
				"secretName": smt.SecretName,
			}).
			SetMultipartField("file", smt.SecretName, "application/octet-stream", data).
			Post(r.blobDataRelaitiveURL)
		if err != nil {
			return fmt.Errorf("ошибка запроса: %w", err)
		}
		if resp.IsStatusFailure() {
			return fmt.Errorf("сервер вернул ошибку: %s", resp.Status())
		}
		return nil
	default:
		dataBytes, err := io.ReadAll(data)
		defer data.Close()

		if err != nil {
			return errors.New("ошибка чтения данных секрета")
		}
		req := createSecretDataReq{
			SecretName: smt.SecretName,
			SecretType: smt.SecretType.ToString(),
			Data:       dataBytes,
		}
		resp, err := r.restiClient.R().
			SetContext(ctx).
			SetHeader("content-type", "application/json").
			SetBody(req).
			Post(r.textDataRelaitiveURL)
		if err != nil {
			return fmt.Errorf("ошибка запроса: %w", err)
		}
		if resp.IsStatusFailure() {
			return fmt.Errorf("сервер вернул ошибку: %s", resp.Status())
		}
		return nil
	}
}

func (r *restySecretsDataRepo) GetSecretData(ctx context.Context, smt secretsDomain.SecretsMetadata) (io.ReadCloser, error) {
	if smt.SecretType == secretsDomain.SecretTypeBinary {
		targetURL := r.blobDataRelaitiveURL
		resp, err := r.restiClient.R().
			SetContext(ctx).
			SetHeader("X-Secret-Name", smt.SecretName).
			SetHeader("X-Secret-Version", strconv.Itoa(int(smt.Version))).
			SetResponseDoNotParse(true).
			Get(targetURL)

		if err != nil {
			return nil, fmt.Errorf("ошибка запроса: %w", err)
		}

		if resp.IsStatusFailure() {
			return nil, fmt.Errorf("сервер вернул ошибку: %s", resp.Status())
		}

		return resp.Body, nil
	} else {
		targetURL := r.textDataRelaitiveURL
		dataResp := getSecretDataResp{}
		resp, err := r.restiClient.R().
			SetContext(ctx).
			SetHeader("X-Secret-Name", smt.SecretName).
			SetHeader("X-Secret-Version", strconv.Itoa(int(smt.Version))).
			SetResult(&dataResp).
			Get(targetURL)

		if err != nil {
			return nil, fmt.Errorf("ошибка запроса: %w", err)
		}

		if resp.IsStatusFailure() {
			return nil, fmt.Errorf("сервер вернул ошибку: %s", resp.Status())
		}

		return io.NopCloser(bytes.NewReader(dataResp.Data)), nil
	}

}
