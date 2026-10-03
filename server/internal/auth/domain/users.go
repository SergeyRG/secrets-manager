package domain

import (
	"errors"
	"regexp"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
)

var ErrInvalidLoginFormat error = errors.New("неверный формат логина")

type User struct {
	UserID  domain.UserID `json:"-"`
	Login   string        `json:"login"`
	PwdHash string        `json:"password"`
}

var alphaNumericUnderscoreRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func (u *User) ValidateLogin() error {
	if !alphaNumericUnderscoreRegex.MatchString(u.Login) {
		return ErrInvalidLoginFormat
	}
	return nil
}

func NewUser(Login string, PwdHash string) (User, error) {
	user := User{
		UserID:  domain.NewUserID(),
		Login:   Login,
		PwdHash: PwdHash,
	}

	if err := user.ValidateLogin(); err != nil {
		return User{}, err
	}

	return user, nil
}
