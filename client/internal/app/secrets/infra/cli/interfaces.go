package cli

import (
	"io"

	"github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	sharedCli "github.com/SergeyRG/secrets-manager/client/internal/app/shared/infra/cli"
)

type View interface {
	ProvideDataToUser(domain.SecretsMetadata, io.ReadCloser, sharedCli.Prompter) error
}

type SecretDataGetter interface {
	GetDataFromUser(sharedCli.Prompter) (io.ReadCloser, error)
}
