package mem

import (
	"context"
	"errors"
)

type MemPasswordProvider struct {
	passwd string
}

func NewMemPasswordProvider(passwd string) *MemPasswordProvider {
	return &MemPasswordProvider{passwd: passwd}
}

func (p *MemPasswordProvider) GetPassword(ctx context.Context) (psswd string, err error) {
	return p.passwd, nil
}

func (p *MemPasswordProvider) CreatePassword(ctx context.Context) (psswd string, err error) {
	return "", errors.New("неприменимо")
}
