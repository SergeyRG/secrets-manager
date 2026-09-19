package cli

type Prompter interface {
	Send(message string) error
	Receive(secure bool) (message string, err error)
}
