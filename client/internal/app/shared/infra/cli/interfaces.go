//go:generate mockgen -source=$GOFILE -destination=mocks/mocks.go -package=mocks

package cli

type Prompter interface {
	Send(message string) error
	Receive(secure bool) (message string, err error)
}
