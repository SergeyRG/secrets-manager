package usecases

import "context"

type Command[Input any, Output any] interface {
	Execute(ctx context.Context, input Input) (Output, error)
}
