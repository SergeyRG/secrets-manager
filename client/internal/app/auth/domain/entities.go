package domain

type TokenStorage interface {
	GetToken() string
	SetToken() string
}
