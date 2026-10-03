package usecases

import "errors"

var (
	ErrEncryptionKeyDoesntExists error = errors.New("ключ шифрования для пользователя не установлен")
)
