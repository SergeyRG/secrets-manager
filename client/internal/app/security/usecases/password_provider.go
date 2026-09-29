//go:generate mockgen -source=$GOFILE -destination=mocks/mocks.go -package=mocks

package usecases

import "context"

type PasswdProvider interface {
	GetPassword(context.Context) (string, error)
	CreatePassword(context.Context) (string, error)
}
