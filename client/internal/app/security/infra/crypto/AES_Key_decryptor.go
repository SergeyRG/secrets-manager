package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"

	"golang.org/x/crypto/argon2"
)

func DecryptKeyWithPassword(encryptedBlock []byte, password string) ([]byte, error) {
	const saltSize = 16
	const nonceSize = 12

	if len(encryptedBlock) < saltSize+nonceSize+16 {
		return nil, fmt.Errorf("crypto: invalid encrypted block size")
	}

	salt := encryptedBlock[:saltSize]
	nonce := encryptedBlock[saltSize : saltSize+nonceSize]
	ciphertext := encryptedBlock[saltSize+nonceSize:]

	kek := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	mainKey, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: invalid password or corrupted data")
	}

	return mainKey, nil
}
