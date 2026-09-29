package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
)

const (
	chunkSize          = 64 * 1024
	tagSize            = 16
	encryptedChunkSize = chunkSize + tagSize
	streamNonceSize    = 12
)

var (
	ErrVerificationFailed = errors.New("crypto: ошибка верификации (данные испорчены или неверный ключ)")
	ErrInvalidKeySize     = errors.New("crypto: неправильная длина ключа (должен быть 16, 24, или 32 байта)")
)

func EncryptStream(key []byte, plainStream io.ReadCloser) (io.ReadCloser, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrInvalidKeySize
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()

	go func() {
		defer plainStream.Close()
		defer pw.Close()

		mainNonce := make([]byte, streamNonceSize)
		if _, err := io.ReadFull(rand.Reader, mainNonce); err != nil {
			_ = pw.CloseWithError(err)
			return
		}

		if _, err := pw.Write(mainNonce); err != nil {
			return
		}

		buffer := make([]byte, chunkSize)
		var chunkIndex uint64 = 0

		for {
			n, readErr := io.ReadFull(plainStream, buffer)
			isLast := readErr != nil
			if n > 0 {
				chunkNonce := deriveNonce(mainNonce, chunkIndex)

				var ad []byte
				if isLast {
					ad = []byte{1}
				} else {
					ad = []byte{0}
				}

				encryptedData := aesGCM.Seal(nil, chunkNonce, buffer[:n], ad)

				if _, writeErr := pw.Write(encryptedData); writeErr != nil {
					return
				}
				chunkIndex++
			}

			if readErr != nil {
				if errors.Is(readErr, io.EOF) || errors.Is(readErr, io.ErrUnexpectedEOF) {
					return
				}
				_ = pw.CloseWithError(readErr)
				return
			}
		}
	}()

	return pr, nil
}

func DecryptStream(key []byte, encryptedStream io.ReadCloser) (io.ReadCloser, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrInvalidKeySize
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()

	go func() {
		defer encryptedStream.Close()
		defer pw.Close()

		mainNonce := make([]byte, streamNonceSize)
		if _, err := io.ReadFull(encryptedStream, mainNonce); err != nil {
			_ = pw.CloseWithError(err)
			return
		}

		buffer := make([]byte, encryptedChunkSize)
		var chunkIndex uint64 = 0
		var lastChunkVerified bool

		for {
			n, readErr := io.ReadFull(encryptedStream, buffer)
			if n > 0 {
				chunkNonce := deriveNonce(mainNonce, chunkIndex)

				isLast := readErr != nil

				var ad []byte
				if isLast {
					ad = []byte{1}
				} else {
					ad = []byte{0}
				}

				decryptedData, err := aesGCM.Open(nil, chunkNonce, buffer[:n], ad)
				if err != nil {
					_ = pw.CloseWithError(ErrVerificationFailed)
					return
				}

				if isLast {
					lastChunkVerified = true
				}

				if _, writeErr := pw.Write(decryptedData); writeErr != nil {
					return
				}
				chunkIndex++
			}

			if readErr != nil {
				if errors.Is(readErr, io.EOF) || errors.Is(readErr, io.ErrUnexpectedEOF) {
					if !lastChunkVerified {
						_ = pw.CloseWithError(ErrVerificationFailed)
						return
					}
					return
				}
				_ = pw.CloseWithError(readErr)
				return
			}
		}
	}()
	return pr, nil
}

func deriveNonce(mainNonce []byte, index uint64) []byte {
	nonce := make([]byte, len(mainNonce))
	copy(nonce, mainNonce)

	counter := binary.BigEndian.Uint64(nonce[4:])
	counter ^= index
	binary.BigEndian.PutUint64(nonce[4:], counter)

	return nonce
}
