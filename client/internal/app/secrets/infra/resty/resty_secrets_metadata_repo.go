package resty

import (
	"context"
	"fmt"
	"iter"
	"strconv"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	"resty.dev/v3"
)

type getUserSecretMetadataResp struct {
	SecretName string `json:"secret_name"`
	Version    int64  `json:"version"`
	SecretType string `json:"secret_type"`
}

type restySecretsRepo struct {
	restiClient          *resty.Client
	metadataRelaitiveURL string
}

func NewRestySecretsRepo(restiClient *resty.Client, metadataRelaitiveURL string) *restySecretsRepo {
	return &restySecretsRepo{restiClient: restiClient, metadataRelaitiveURL: metadataRelaitiveURL}
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
		Get(r.metadataRelaitiveURL)

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

func (r *restySecretsRepo) GetUserSecretsMetadataPage(ctx context.Context, page int, perPage int) iter.Seq2[secretsDomain.SecretsMetadata, error] {
	return nil
}
