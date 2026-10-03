package cli_test

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/SergeyRG/secrets-manager/client/internal/app/secrets/infra/cli"

	//mocks "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/infra/cli/mocks"
	sharedCli "github.com/SergeyRG/secrets-manager/client/internal/app/shared/infra/cli"
	mocks "github.com/SergeyRG/secrets-manager/client/internal/app/shared/infra/cli/mocks"
	"github.com/stretchr/testify/assert"

	"go.uber.org/mock/gomock"
)

type ManualMockSecretDataGetter struct {
	ctrl     *gomock.Controller
	recorder *ManualMockSecretDataGetterRecorder
}
type ManualMockSecretDataGetterRecorder struct {
	mock *ManualMockSecretDataGetter
}

func NewManualMockSecretDataGetter(ctrl *gomock.Controller) *ManualMockSecretDataGetter {
	mock := &ManualMockSecretDataGetter{ctrl: ctrl}
	mock.recorder = &ManualMockSecretDataGetterRecorder{mock}
	return mock
}
func (m *ManualMockSecretDataGetter) EXPECT() *ManualMockSecretDataGetterRecorder { return m.recorder }

func (m *ManualMockSecretDataGetter) GetDataFromUser(prompter sharedCli.Prompter) (io.ReadCloser, error) {
	ret := m.ctrl.Call(m, "GetDataFromUser", prompter)

	var ret0 io.ReadCloser
	if ret[0] != nil {
		ret0 = ret[0].(io.ReadCloser)
	}

	var ret1 error
	if ret[1] != nil {
		ret1 = ret[1].(error)
	}

	return ret0, ret1
}

func (mr *ManualMockSecretDataGetterRecorder) GetDataFromUser(prompter any) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDataFromUser", reflect.TypeOf((*ManualMockSecretDataGetter)(nil).GetDataFromUser), prompter)
}

func TestCreateSecretCommandHandler_Handle(t *testing.T) {
	secretName := "my_binary_vault"

	t.Run("Ошибка: Неверное количество аргументов CLI", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPrompter := mocks.NewMockPrompter(ctrl)
		h := cli.NewCreateSecretDataCommandHandler(nil, nil)

		err := h.Handle(context.Background(), []string{}, mockPrompter)
		assert.ErrorIs(t, err, sharedCli.ErrInvalidArguments)
	})

	t.Run("Ошибка: Передано пустое имя секрета", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPrompter := mocks.NewMockPrompter(ctrl)
		h := cli.NewCreateSecretDataCommandHandler(nil, nil)

		err := h.Handle(context.Background(), []string{""}, mockPrompter)
		assert.ErrorIs(t, err, sharedCli.ErrInvalidArguments)
	})

	t.Run("Интерактивное меню - Сбой вывода подсказки меню", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPrompter := mocks.NewMockPrompter(ctrl)

		mockPrompter.EXPECT().
			Send(gomock.Regex("Выберите тип секрета:")).
			Return(errors.New("stdout crash")).
			Times(1)

		h := cli.NewCreateSecretDataCommandHandler(nil, nil)

		err := h.Handle(context.Background(), []string{secretName}, mockPrompter)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка вывода данных")
	})

	t.Run("Интерактивное меню - Сбой чтения ввода пользователя", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPrompter := mocks.NewMockPrompter(ctrl)

		mockPrompter.EXPECT().Send(gomock.Any()).Return(nil).Times(1)
		mockPrompter.EXPECT().Receive(false).Return("", errors.New("stdin EOF")).Times(1)

		h := cli.NewCreateSecretDataCommandHandler(nil, nil)

		err := h.Handle(context.Background(), []string{secretName}, mockPrompter)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка ввода данных")
	})
}
