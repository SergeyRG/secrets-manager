package usecases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/SergeyRG/secrets-manager/client/internal/app/auth/usecases"
	"github.com/SergeyRG/secrets-manager/client/internal/app/auth/usecases/mocks"
	"go.uber.org/mock/gomock"
)

func TestLoginUseCase_Execute(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthClient := mocks.NewMockAuthClient(ctrl)
	mockTokenStorage := mocks.NewMockTokenStorage(ctrl)

	uc := usecases.NewLoginUseCase(mockAuthClient)
	ctx := context.Background()

	type mockBehavior func()
	tests := []struct {
		name         string
		mockBehavior mockBehavior
		wantLogin    string
		wantErr      error
	}{
		{
			name: "Успешная аутентификация",
			mockBehavior: func() {
				mockAuthClient.EXPECT().
					Authenticate(ctx, mockTokenStorage).
					Return("sergey_rg", nil).
					Times(1)
			},
			wantLogin: "sergey_rg",
			wantErr:   nil,
		},
		{
			name: "Ошибка: Сервер недоступен",
			mockBehavior: func() {
				mockAuthClient.EXPECT().
					Authenticate(ctx, mockTokenStorage).
					Return("", usecases.ErrServerUnavailable).
					Times(1)
			},
			wantLogin: "",
			wantErr:   usecases.ErrServerUnavailable,
		},
		{
			name: "Ошибка: Неверный логин или пароль",
			mockBehavior: func() {
				mockAuthClient.EXPECT().
					Authenticate(ctx, mockTokenStorage).
					Return("", usecases.ErrAuthenticationFailed).
					Times(1)
			},
			wantLogin: "",
			wantErr:   usecases.ErrAuthenticationFailed,
		},
		{
			name: "Неожиданная системная ошибка",
			mockBehavior: func() {
				mockAuthClient.EXPECT().
					Authenticate(ctx, mockTokenStorage).
					Return("", errors.New("internal database error")).
					Times(1)
			},
			wantLogin: "",
			wantErr:   errors.New("internal database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockBehavior()

			login, err := uc.Execute(ctx, mockTokenStorage)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("ожидалась ошибка %v, но вернулся nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					t.Errorf("получена ошибка: %v, ожидалась: %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("не ожидалось ошибки, но получена: %v", err)
			}

			if login != tt.wantLogin {
				t.Errorf("получен логин: %q, ожидался: %q", login, tt.wantLogin)
			}
		})
	}
}
