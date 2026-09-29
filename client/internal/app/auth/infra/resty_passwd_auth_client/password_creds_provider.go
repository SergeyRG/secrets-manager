//go:generate mockgen -source=$GOFILE -destination=mocks/mocks.go -package=mockspackage usecases

package authclient

import (
	"context"
)

type PasswdCreds struct {
	Login  string
	Passwd string
}

type PasswdCredsProvider interface {
	GetCreds(context.Context) (PasswdCreds, error)
}
