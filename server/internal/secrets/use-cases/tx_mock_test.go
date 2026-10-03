package usecases_test

import "context"

type MockTxManager struct{}

func (m *MockTxManager) WithinTransaction(ctx context.Context, f func(txCtx context.Context) error) error {
	return f(ctx)
}
