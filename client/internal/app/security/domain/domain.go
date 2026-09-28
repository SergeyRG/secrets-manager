package domain

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

var (
	ErrKeyEmpty = errors.New("ключ шифрования не задан/не расшифрован")
)

const KEY_SIZE int = 32

type KeyEncryptor func(data []byte, password string) ([]byte, error)
type KeyDecryptor func(encryptedBlock []byte, password string) ([]byte, error)

type StreamEncryptor func(key []byte, plain io.ReadCloser) (io.ReadCloser, error)
type StreamDecryptor func(key []byte, encrypted io.ReadCloser) (io.ReadCloser, error)

type KeyStorage struct {
	key             []byte
	keyEncryptor    KeyEncryptor
	keyDecryptor    KeyDecryptor
	streamEncryptor StreamEncryptor
	streamDecryptor StreamDecryptor
}

func NewKeyStorage(
	kd KeyDecryptor,
	ke KeyEncryptor,
	se StreamEncryptor,
	sd StreamDecryptor,
) *KeyStorage {
	return &KeyStorage{
		key:             nil,
		keyEncryptor:    ke,
		keyDecryptor:    kd,
		streamEncryptor: se,
		streamDecryptor: sd,
	}
}

func (ks *KeyStorage) GenerateNewKey() error {
	key := make([]byte, KEY_SIZE)

	_, err := io.ReadFull(rand.Reader, key)
	if err != nil {
		return fmt.Errorf("ошибка генерации случайных байт: %w", err)
	}

	ks.key = key
	return nil
}

func (ks *KeyStorage) LoadFromEncrypted(encryptedBlock []byte, password string) error {
	decryptedKey, err := ks.keyDecryptor(encryptedBlock, password)
	if err != nil {
		return fmt.Errorf("ошибка расшифрования мастер-ключа: %w", err)
	}
	ks.key = decryptedKey
	return nil
}

func (ks *KeyStorage) GetEncryptedKey(password string) ([]byte, error) {
	if ks.key == nil {
		return nil, ErrKeyEmpty
	}
	return ks.keyEncryptor(ks.key, password)
}
func (ks *KeyStorage) EncryptData(plainStream io.ReadCloser) (io.ReadCloser, error) {
	if ks.key == nil {
		return nil, ErrKeyEmpty
	}
	return ks.streamEncryptor(ks.key, plainStream)
}

func (ks *KeyStorage) DecryptData(encryptedStream io.ReadCloser) (io.ReadCloser, error) {
	if ks.key == nil {
		return nil, ErrKeyEmpty
	}
	return ks.streamDecryptor(ks.key, encryptedStream)
}
