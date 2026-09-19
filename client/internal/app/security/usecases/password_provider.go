package usecases

import "context"

type PasswdProvider interface {
	GetPassword(context.Context) (string, error)
	CreatePassword(context.Context) (string, error)
}
