package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"

	"golang.org/x/crypto/argon2"
)

func EncryptKeyWithPassword(mainKey []byte, password string) ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	kek := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	encryptedKey := aesGCM.Seal(nil, nonce, mainKey, nil)

	result := append(salt, nonce...)
	result = append(result, encryptedKey...)

	return result, nil
}
