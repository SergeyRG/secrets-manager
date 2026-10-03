package tokenstorage

import "context"

type JWTTokenStorage struct {
	storage []byte
}

func NewJWTTokenStorage(storage []byte) *JWTTokenStorage {
	return &JWTTokenStorage{storage: storage}
}

func (tk *JWTTokenStorage) LoadRaw(ctx context.Context) ([]byte, error) {
	return tk.storage, nil
}

func (tk *JWTTokenStorage) SaveRaw(ctx context.Context, data []byte) error {
	tk.storage = data
	return nil
}
