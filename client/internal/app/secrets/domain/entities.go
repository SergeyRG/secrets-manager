package domain

import (
	"errors"
	"strings"
)

var (
	ErrEmptySecretName   = errors.New("имя секрета не может быть пустым")
	ErrInvalidType       = errors.New("указан неверный тип секрета")
	ErrInvalidNameFormat = errors.New("имя может содержать только латинские буквы, цифры и нижнее подчеркивание")
)

type SecretType int64

const (
	SecretTypeUnknown SecretType = iota
	SecretTypeBinary
	SecretTypeFreeText
	SecretTypeAuthData
	SecretTypeBankCard

	SecretTypeMax
)

func (stp SecretType) ToString() string {
	var str string

	switch stp {
	case SecretTypeBinary:
		str = "SECRET_TYPE_BINARY"
	case SecretTypeFreeText:
		str = "SECRET_TYPE_FREE_TEXT"
	case SecretTypeAuthData:
		str = "SECRET_TYPE_AUTH_DATA"
	case SecretTypeBankCard:
		str = "SECRET_TYPE_BANK_CARD"
	default:
		str = "SECRET_TYPE_UNKNOWN"
	}

	return str
}

func SecretTypeFromString(str string) SecretType {
	var st SecretType

	switch strings.ToUpper(str) {
	case "SECRET_TYPE_BINARY":
		st = SecretTypeBinary
	case "SECRET_TYPE_FREE_TEXT":
		st = SecretTypeFreeText
	case "SECRET_TYPE_AUTH_DATA":
		st = SecretTypeAuthData
	case "SECRET_TYPE_BANK_CARD":
		st = SecretTypeBankCard
	default:
		st = SecretTypeUnknown
	}
	return st
}

type SecretsMetadata struct {
	SecretName string
	SecretType SecretType
	Version    int64
}

//var alphaNumericUnderscoreRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// func validateName(name string) error {
// 	if !alphaNumericUnderscoreRegex.MatchString(name) {
// 		return ErrInvalidNameFormat
// 	}
// 	return nil
// }

// func NewSecretMetadata(stp SecretType, name string) (SecretsMetadata, error) {
// 	if name == "" {
// 		return SecretsMetadata{}, ErrEmptySecretName
// 	}

// 	if validateName(name) != nil {
// 		return SecretsMetadata{}, ErrInvalidNameFormat
// 	}

// 	if stp <= SecretTypeUnknown || stp >= SecretTypeMax {
// 		return SecretsMetadata{}, ErrInvalidType
// 	}

// 	m := SecretsMetadata{
// 		SecretName: name,
// 		SecretType: stp,
// 	}
// 	return m, nil
// }

// type SecretTextData interface {
// 	ToString() string
// }

// type SecretAuthData struct {
// 	Login    string
// 	Password string
// }

// func (s SecretAuthData) ToString() string {
// 	return fmt.Sprintf("login: %s\n password: %s\n", s.Login, s.Password)
// }

// type SecretBankData struct {
// 	CardNum string
// 	CVC     string
// }

// func (s SecretBankData) ToString() string {
// 	return fmt.Sprintf("Card number: %s\n CVC: %s\n", s.CardNum, s.CVC)
// }

// type SecretFreeText struct {
// 	data string
// }

// func (s SecretFreeText) ToString() string {
// 	return fmt.Sprintf("data: %s\n CVC: %s\n", s.data)
// }

// type SecretBlobData struct {
// 	SecretType SecretType
// 	Data       io.ReadCloser
// }
