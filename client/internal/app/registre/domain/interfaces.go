//go:generate mockgen -source=$GOFILE -destination=mocks/mocks.go -package=mocks
package domain

import "context"

type RegistreProvider interface {
	Registre(context.Context, User) error
}

type UserDataProvider interface {
	GetUserData(context.Context) (User, error)
}
