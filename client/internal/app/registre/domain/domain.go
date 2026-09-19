package domain

import (
	"errors"
	"regexp"
)

var ErrInvalidLoginFormat error = errors.New("неверный формат логина")
var ErrTooSimplePassword error = errors.New("слишком простой пароль")

type User struct {
	Login  string
	Passwd string
}

func NewUser(login string, passwd string) (User, error) {
	err := ValidateLogin(login)
	if err != nil {
		return User{}, err
	}

	if len(passwd) < 6 {
		return User{}, ErrTooSimplePassword
	}

	return User{Login: login, Passwd: passwd}, nil
}

var alphaNumericUnderscoreRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func ValidateLogin(login string) error {
	if !alphaNumericUnderscoreRegex.MatchString(login) {
		return ErrInvalidLoginFormat
	}
	return nil
}
