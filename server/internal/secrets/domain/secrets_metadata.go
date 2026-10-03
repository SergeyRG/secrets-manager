package domain

import (
	"crypto/rand"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
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
	UserID       domain.UserID
	SecretID     domain.SecretID
	SecretName   string
	SecretType   SecretType
	TimeCreation time.Time
	Version      int64
	VersionID    domain.SecretVersionID
}

var alphaNumericUnderscoreRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func ValidateName(name string) error {
	if !alphaNumericUnderscoreRegex.MatchString(name) {
		return ErrInvalidNameFormat
	}
	return nil
}

func NewSecretMetadata(uID domain.UserID, stp SecretType, name string) (SecretsMetadata, error) {
	if name == "" {
		return SecretsMetadata{}, ErrEmptySecretName
	}

	if ValidateName(name) != nil {
		return SecretsMetadata{}, ErrInvalidNameFormat
	}

	if stp <= SecretTypeUnknown || stp >= SecretTypeMax {
		return SecretsMetadata{}, ErrInvalidType
	}

	sID := domain.SecretID(rand.Text())
	svID := domain.SecretVersionID(rand.Text())

	m := SecretsMetadata{
		UserID:       uID,
		SecretID:     sID,
		SecretName:   name,
		SecretType:   stp,
		VersionID:    svID,
		Version:      1,
		TimeCreation: time.Now().UTC(),
	}
	return m, nil
}

func NewSecretVersionMetadata(sm SecretsMetadata) (SecretsMetadata, error) {
	sm.VersionID = domain.SecretVersionID(rand.Text())
	sm.Version = sm.Version + 1
	sm.TimeCreation = time.Now().UTC()

	return sm, nil
}
