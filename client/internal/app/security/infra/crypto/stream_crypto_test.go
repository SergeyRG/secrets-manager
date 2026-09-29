package crypto_test

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"testing"

	"github.com/SergeyRG/secrets-manager/client/internal/app/security/infra/crypto"
)

func TestEncryptDecryptStream_Integration(t *testing.T) {
	validKey := make([]byte, 32) // AES-256
	_, _ = io.ReadFull(rand.Reader, validKey)

	smallData := []byte("hello world, crypto stream test!")
	exactChunkData := make([]byte, 64*1024) // Ровно 1 чанк
	largeData := make([]byte, 128*1024+15)  // Больше 2 чанков с остатком
	_, _ = io.ReadFull(rand.Reader, exactChunkData)
	_, _ = io.ReadFull(rand.Reader, largeData)

	tests := []struct {
		name string
		data []byte
	}{
		{name: "Данные меньше размера чанка", data: smallData},
		{name: "Данные ровно с размер чанка (64 КБ)", data: exactChunkData},
		{name: "Данные больше нескольких чанков (~130 КБ)", data: largeData},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plainIn := io.NopCloser(bytes.NewReader(tt.data))
			encOut, err := crypto.EncryptStream(validKey, plainIn)
			if err != nil {
				t.Fatalf("ошибка шифрования стрима: %v", err)
			}

			decOut, err := crypto.DecryptStream(validKey, encOut)
			if err != nil {
				t.Fatalf("ошибка расшифрования стрима: %v", err)
			}

			result, err := io.ReadAll(decOut)
			if err != nil {
				t.Fatalf("ошибка чтения расшифрованного потока: %v", err)
			}
			_ = decOut.Close()

			if !bytes.Equal(result, tt.data) {
				t.Error("расшифрованные данные не совпадают с исходными")
			}
		})
	}
}

func TestStreamCrypto_InvalidKeySize(t *testing.T) {
	invalidKey := []byte("bad-key") // Неверная длина для AES
	plainIn := io.NopCloser(bytes.NewReader([]byte("test")))

	_, err := crypto.EncryptStream(invalidKey, plainIn)
	if !errors.Is(err, crypto.ErrInvalidKeySize) {
		t.Errorf("ожидалась ошибка ErrInvalidKeySize при шифровании, получено: %v", err)
	}

	_, err = crypto.DecryptStream(invalidKey, plainIn)
	if !errors.Is(err, crypto.ErrInvalidKeySize) {
		t.Errorf("ожидалась ошибка ErrInvalidKeySize при расшифровании, получено: %v", err)
	}
}

func TestDecryptStream_TamperedData(t *testing.T) {
	key := make([]byte, 32)
	_, _ = io.ReadFull(rand.Reader, key)
	payload := []byte("секретные данные, которые попытаются подменить")

	encOut, err := crypto.EncryptStream(key, io.NopCloser(bytes.NewReader(payload)))
	if err != nil {
		t.Fatalf("ошибка шифрования: %v", err)
	}

	encryptedBytes, err := io.ReadAll(encOut)
	if err != nil {
		t.Fatalf("ошибка вычитки зашифрованных данных: %v", err)
	}

	if len(encryptedBytes) > 20 {
		encryptedBytes[20] ^= 0xFF
	} else {
		t.Skip("слишком короткий зашифрованный поток для модификации")
	}

	tamperedStream := io.NopCloser(bytes.NewReader(encryptedBytes))
	decOut, err := crypto.DecryptStream(key, tamperedStream)
	if err != nil {
		t.Fatalf("метод создания стрима не должен падать, ошибка ожидается при чтении: %v", err)
	}

	_, err = io.ReadAll(decOut)
	if !errors.Is(err, crypto.ErrVerificationFailed) {
		t.Errorf("ожидалась ошибка ErrVerificationFailed, но получена: %v", err)
	}
	_ = decOut.Close()
}
