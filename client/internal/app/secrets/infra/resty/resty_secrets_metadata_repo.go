package resty

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	secretsUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/usecases"
	"resty.dev/v3"
)

type getUserSecretMetadataResp struct {
	SecretName string `json:"secret_name"`
	Version    int64  `json:"version"`
	SecretType string `json:"secret_type"`
}

type restySecretsRepo struct {
	restiClient             *resty.Client
	metadataRelativeURL     string
	metadataListRelativeURL string
}

func NewRestySecretsRepo(
	restiClient *resty.Client,
	metadataRelativeURL string,
	metadaListURL string,
) *restySecretsRepo {
	return &restySecretsRepo{
		restiClient:             restiClient,
		metadataRelativeURL:     metadataRelativeURL,
		metadataListRelativeURL: metadaListURL,
	}
}

func (r *restySecretsRepo) AddSecretMetadata(context.Context, secretsDomain.SecretsMetadata) error {
	return nil
}

func (r *restySecretsRepo) GetUserSecretMetadataByName(ctx context.Context, sName string, version int) (secretsDomain.SecretsMetadata, error) {

	smResp := getUserSecretMetadataResp{}
	resp, err := r.restiClient.R().
		SetContext(ctx).
		SetHeader("X-Secret-Name", sName).
		SetHeader("X-Secret-Version", strconv.Itoa(version)).
		SetResult(&smResp).
		Get(r.metadataRelativeURL)

	if err != nil {
		return secretsDomain.SecretsMetadata{}, fmt.Errorf("ошибка запроса: %w", err)
	}

	if resp.IsStatusFailure() {
		return secretsDomain.SecretsMetadata{}, fmt.Errorf("сервер вернул ошибку: %s", resp.Status())
	}

	return secretsDomain.SecretsMetadata{
		SecretName: smResp.SecretName,
		SecretType: secretsDomain.SecretTypeFromString(smResp.SecretType),
		Version:    smResp.Version,
	}, nil
}

func (r *restySecretsRepo) GetUserSecretsMetadataPage(
	ctx context.Context,
	page int,
	perPage int,
) ([]secretsDomain.SecretsMetadata, error) {

	response, err := r.restiClient.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"page":     strconv.Itoa(page),
			"per_page": strconv.Itoa(perPage),
		}).
		Get(r.metadataListRelativeURL)

	if err != nil {
		return nil, fmt.Errorf("%w: %v", secretsUsecases.ErrServerUnavailable, err)
	}

	if response.IsStatusFailure() {
		return nil, fmt.Errorf("сервер вернул статус ошибки: %s", response.Status())
	}

	var list []secretsDomain.SecretsMetadata
	scanner := bufio.NewScanner(bytes.NewReader(response.Bytes()))

	for scanner.Scan() {
		lineBytes := scanner.Bytes()
		if len(bytes.TrimSpace(lineBytes)) == 0 {
			continue
		}

		var smResp getUserSecretMetadataResp
		if err := json.Unmarshal(lineBytes, &smResp); err != nil {
			return nil, fmt.Errorf("ошибка парсинга строки ndjson: %w", err)
		}
		sm := secretsDomain.SecretsMetadata{
			SecretName: smResp.SecretName,
			SecretType: secretsDomain.SecretTypeFromString(smResp.SecretType),
			Version:    smResp.Version,
		}
		list = append(list, sm)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ошибка обработки потока данных: %w", err)
	}

	return list, nil
}
