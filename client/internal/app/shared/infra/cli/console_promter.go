package cli

import (
	"bufio"
	"fmt"
	"os"

	"golang.org/x/term"
)

type ConsolePrompter struct {
	scanner *bufio.Scanner
}

func NewConsolePrompter() *ConsolePrompter {
	return &ConsolePrompter{
		scanner: bufio.NewScanner(os.Stdin),
	}
}

func (p *ConsolePrompter) Send(msg string) error {
	_, err := fmt.Print(msg)
	return err
}

func (p *ConsolePrompter) Receive(secure bool) (string, error) {
	if secure {
		return p.readSecure()
	}
	return p.readStandard()
}

func (p *ConsolePrompter) readStandard() (string, error) {
	if !p.scanner.Scan() {
		if err := p.scanner.Err(); err != nil {
			return "", err
		}
		return "", os.ErrClosed
	}
	return p.scanner.Text(), nil
}

func (p *ConsolePrompter) readSecure() (string, error) {
	fd := int(os.Stdin.Fd())

	bytePassword, err := term.ReadPassword(fd)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения скрытого ввода: %w", err)
	}

	fmt.Println()

	return string(bytePassword), nil
}
