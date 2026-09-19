package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

type REPL struct {
	router   *CommandRouter
	prompter Prompter
}

func NewREPL(router *CommandRouter, prompter Prompter) *REPL {
	return &REPL{
		router:   router,
		prompter: prompter,
	}
}

func (r *REPL) Start(ctx context.Context) {
	r.prompter.Send("Добро пожаловать в Secrets Manager CLI!\n")
	r.prompter.Send("Введите 'help' для просмотра команд, 'quit' для выхода.\n\n")

	for {
		select {
		case <-ctx.Done():
			r.prompter.Send("\nСессия завершена внешним сигналом.\n")
			return
		default:
			if ctx.Err() != nil {
				return
			}
			r.prompter.Send("secrets-cli> ")

			input, err := r.prompter.Receive(false)
			if err != nil {
				if errors.Is(os.ErrClosed, err) {
					return
				}
				r.prompter.Send(fmt.Sprintf("Ошибка ввода: %v\n", err))
				continue
			}

			input = strings.TrimSpace(input)
			if input == "" {
				continue
			}

			if input == "quit" {
				r.prompter.Send("Завершение приложения!\n")
				return
			}

			err = r.router.HandleCommand(ctx, input, r.prompter)
			if err != nil {
				r.prompter.Send(fmt.Sprintf("Ошибка выполнения команды: %v\n", err))
				continue
			}

			//			r.prompter.Send(response)
		}
	}
}
