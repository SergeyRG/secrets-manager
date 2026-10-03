package crypto_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/SergeyRG/secrets-manager/client/internal/app/security/infra/crypto"
)

func TestPasswordCrypto_Roundtrip(t *testing.T) {
	password := "my-strong-p@ssword-2026"
	secretData := []byte("this-is-a-32-byte-master-key-xyz")

	validBlock, err := crypto.EncryptDataWithPassword(secretData, password)
	if err != nil {
		t.Fatalf("EncryptDataWithPassword() вернул ошибку: %v", err)
	}

	expectedMinSize := 16 + 12 + len(secretData) + 16
	if len(validBlock) != expectedMinSize {
		t.Errorf("неверный размер зашифрованного блока: получено %d, ожидалось %d", len(validBlock), expectedMinSize)
	}

	t.Run("Успешное расшифрование с правильным паролем", func(t *testing.T) {
		// 2. Тестируем расшифрование вашей функцией
		decrypted, err := crypto.DecryptDataWithPassword(validBlock, password)
		if err != nil {
			t.Fatalf("не ожидалось ошибки, но получена: %v", err)
		}

		if !bytes.Equal(decrypted, secretData) {
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
		shortBlock := make([]byte, 10) // Меньше минимальных 16+12+16 байт
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

		corruptedBlock[len(corruptedBlock)-1] = corruptedBlock[len(corruptedBlock)-1] ^ 0xFF

		_, err := crypto.DecryptDataWithPassword(corruptedBlock, password)
		if err == nil {
			t.Fatal("ожидалась ошибка GCM верификации для испорченных данных, но получен nil")
		}

		if !strings.Contains(err.Error(), "invalid password or corrupted data") {
			t.Errorf("ожидался текст ошибки про поврежденные данные, получено: %v", err)
		}
	})
}
