package usecases

import (
	"context"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/token_storage.go -package=mocks
type TokenStorage interface {
	LoadRaw(ctx context.Context) ([]byte, error)
	SaveRaw(ctx context.Context, data []byte) error
}

//go:generate mockgen -source=$GOFILE -destination=mocks/auth_client.go -package=mocks
type AuthClient interface {
	Authenticate(ctx context.Context, storage TokenStorage) (login string, err error)
}
