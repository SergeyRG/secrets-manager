package domain

import "crypto/rand"

type UserID string
type SecretID string
type FileID string
type SecretVersionID string

func NewUserID() UserID {
	return UserID(rand.Text())
}

func NewSecretID() SecretID {
	return SecretID(rand.Text())
}

func NewSecretVersionID() SecretVersionID {
	return SecretVersionID(rand.Text())
}
