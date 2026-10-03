//go:generate mockgen -source=$GOFILE -destination=mocks/mocks.go -package=mocks
package usecases

import (
	"context"
	"io"

	"github.com/SergeyRG/secrets-manager/server/internal/files/domain"
)

type FilesRepository interface {
	AddFile(context.Context, domain.FileID, io.ReadCloser) error
	GetFile(context.Context, domain.FileID) (io.ReadCloser, error)
}

type FilesStateRepository interface {
	AddFileState(context.Context, domain.FileID, domain.FileState) error
	GetFileState(context.Context, domain.FileID) (domain.FileState, error)
}
