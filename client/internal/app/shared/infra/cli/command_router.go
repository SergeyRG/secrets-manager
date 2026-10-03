package cli

import (
	"context"
	"fmt"
	"strings"
)

type CommandHandler interface {
	Handle(ctx context.Context, args []string, promter Prompter) error
}

type CommandInfo struct {
	Name        string
	Description string
}

type CommandRegistryEntry struct {
	CommandInfo CommandInfo
	Handler     CommandHandler
}

type CommandRouter struct {
	registry map[string]CommandRegistryEntry
}

func NewCommandRouter(registry map[string]CommandRegistryEntry) *CommandRouter {
	return &CommandRouter{registry: registry}
}

func (r *CommandRouter) HandleCommand(ctx context.Context, line string, promter Prompter) error {
	args := strings.Fields(line)
	if len(args) == 0 {
		return nil
	}

	cmdName := args[0]
	cmdArgs := args[1:]

	if cmdName == "help" {
		return r.getHelpText(promter)
	}

	cmd, exists := r.registry[cmdName]
	if !exists {
		return fmt.Errorf("неизвестная команда '%s'. Введите 'help'", cmdName)
	}

	return cmd.Handler.Handle(ctx, cmdArgs, promter)
}

func (r *CommandRouter) getHelpText(promter Prompter) error {
	promter.Send("Доступные команды:\n")
	promter.Send("  help         - Показать эту справку\n")
	for _, cmd := range r.registry {
		promter.Send(fmt.Sprintf("  %-12s - %s\n", cmd.CommandInfo.Name, cmd.CommandInfo.Description))
	}
	return nil
}
