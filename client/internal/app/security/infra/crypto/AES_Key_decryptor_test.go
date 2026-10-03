package crypto_test

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"strings"
	"testing"

	"github.com/SergeyRG/secrets-manager/client/internal/app/security/infra/crypto"
	"golang.org/x/crypto/argon2"
)

func encryptDataWithPassword(plainText []byte, password string) ([]byte, error) {
	const saltSize = 16
	const nonceSize = 12

	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
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

	ciphertext := aesGCM.Seal(nil, nonce, plainText, nil)

	encryptedBlock := make([]byte, 0, len(salt)+len(nonce)+len(ciphertext))
	encryptedBlock = append(encryptedBlock, salt...)
	encryptedBlock = append(encryptedBlock, nonce...)
	encryptedBlock = append(encryptedBlock, ciphertext...)

	return encryptedBlock, nil
}

func TestDecryptDataWithPassword(t *testing.T) {
	password := "my-strong-p@ssword-2026"
	masterKey := []byte("this-is-a-32-byte-master-key-xyz")

	validBlock, err := encryptDataWithPassword(masterKey, password)
	if err != nil {
		t.Fatalf("ошибка подготовки тестовых данных: %v", err)
	}

	t.Run("Успешное расшифрование с правильным паролем", func(t *testing.T) {
		decrypted, err := crypto.DecryptDataWithPassword(validBlock, password)
		if err != nil {
			t.Fatalf("не ожидалось ошибки, но получена: %v", err)
		}

		if !bytes.Equal(decrypted, masterKey) {
			t.Errorf("расшифрованный ключ не совпадает с оригиналом: %s", string(decrypted))
		}
	})

	t.Run("Ошибка: передан неверный пароль", func(t *testing.T) {
		_, err := crypto.DecryptDataWithPassword(validBlock, "wrong-password")
		if err == nil {
			t.Fatal("ожидалась ошибка расшифрования, но получен nil")
		}

		if !strings.Contains(err.Error(), "invalid password or corrupted data") {
			t.Errorf("ожидался текст ошибки про неверный пароль, получено: %v", err)
		}
	})

	t.Run("Ошибка: слишком короткий размер зашифрованного блока", func(t *testing.T) {
		shortBlock := make([]byte, 10)
		_, err := crypto.DecryptDataWithPassword(shortBlock, password)
		if err == nil {
			t.Fatal("ожидалась ошибка валидации размера, но получен nil")
		}

		if !strings.Contains(err.Error(), "invalid encrypted block size") {
			t.Errorf("ожидался текст ошибки про неверный размер блока, получено: %v", err)
		}
	})

	t.Run("Ошибка: данные блока были повреждены/модифицированы", func(t *testing.T) {
		corruptedBlock := make([]byte, len(validBlock))
		copy(corruptedBlock, validBlock)
		corruptedBlock[len(corruptedBlock)-1] ^= 0xFF

		_, err := crypto.DecryptDataWithPassword(corruptedBlock, password)
		if err == nil {
			t.Fatal("ожидалась ошибка GCM верификации для испорченных данных, но получен nil")
		}

		if !strings.Contains(err.Error(), "invalid password or corrupted data") {
			t.Errorf("ожидался текст ошибки про поврежденные данные, получено: %v", err)
		}
	})
}
