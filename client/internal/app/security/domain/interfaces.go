package domain

import "context"

type EncryptedKeyRepository interface {
	SaveEncryptedKey(ctx context.Context, encKey []byte) error
	GetEncryptedKey(ctx context.Context) (encKey []byte, err error)
}
