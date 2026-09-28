package domain

import (
	"context"
	"errors"
)

var (
	ErrKeyNotFound = errors.New("ключ шифрования не найден")
)

type EncryptedKeyRepository interface {
	SaveEncryptedKey(ctx context.Context, encKey []byte) error
	GetEncryptedKey(ctx context.Context) (encKey []byte, err error)
}
