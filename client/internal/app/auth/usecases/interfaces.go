package usecases

import (
	"context"
)

type TokenStorage interface {
	LoadRaw(ctx context.Context) ([]byte, error)
	SaveRaw(ctx context.Context, data []byte) error
}

type AuthClient interface {
	Authenticate(ctx context.Context, storage TokenStorage) error
}
