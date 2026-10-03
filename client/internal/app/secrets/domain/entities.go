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
