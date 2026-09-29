package usecases_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/SergeyRG/secrets-manager/client/internal/app/security/domain"
	domainMocks "github.com/SergeyRG/secrets-manager/client/internal/app/security/domain/mocks"
	"github.com/SergeyRG/secrets-manager/client/internal/app/security/usecases"
	mocks "github.com/SergeyRG/secrets-manager/client/internal/app/security/usecases/mocks"
	sharedUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/shared/usecases"

	"go.uber.org/mock/gomock"
)

func TestGetUserKeyUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	testPassword := "user_password"
	localKeyBytes := []byte("local_encrypted_key")
	remoteKeyBytes := []byte("remote_encrypted_key")

	tests := []struct {
		name         string
		sessionType  sharedUsecases.SessionType
		mockBehavior func(remote *domainMocks.MockEncryptedKeyRepository, local *domainMocks.MockEncryptedKeyRepository, pp *mocks.MockPasswdProvider)
		wantErr      error
	}{
		{
			name:        "Успешно: Локальная сессия, ключ взят из кэша и расшифрован",
			sessionType: sharedUsecases.SessionTypeLocal,
			mockBehavior: func(remote *domainMocks.MockEncryptedKeyRepository, local *domainMocks.MockEncryptedKeyRepository, pp *mocks.MockPasswdProvider) {
				local.EXPECT().GetEncryptedKey(ctx).Return(localKeyBytes, nil).Times(1)
				pp.EXPECT().GetPassword(ctx).Return(testPassword, nil).Times(1)
			},
			wantErr: nil,
		},
		{
			name:        "Ошибка: Локальная сессия, кэш пуст",
			sessionType: sharedUsecases.SessionTypeLocal,
			mockBehavior: func(remote *domainMocks.MockEncryptedKeyRepository, local *domainMocks.MockEncryptedKeyRepository, pp *mocks.MockPasswdProvider) {
				local.EXPECT().GetEncryptedKey(ctx).Return(nil, domain.ErrKeyNotFound).Times(1)
			},
			wantErr: domain.ErrKeyNotFound,
		},
		{
			name:        "Успешно: Сессия Remote, ключ скачан с сервера, сохранен в кэш и расшифрован",
			sessionType: sharedUsecases.SessionTypeRemote,
			mockBehavior: func(remote *domainMocks.MockEncryptedKeyRepository, local *domainMocks.MockEncryptedKeyRepository, pp *mocks.MockPasswdProvider) {
				local.EXPECT().GetEncryptedKey(ctx).Return(nil, domain.ErrKeyNotFound).Times(1)
				remote.EXPECT().GetEncryptedKey(ctx).Return(remoteKeyBytes, nil).Times(1)
				local.EXPECT().SaveEncryptedKey(ctx, remoteKeyBytes).Return(nil).Times(1)
				pp.EXPECT().GetPassword(ctx).Return(testPassword, nil).Times(1)
			},
			wantErr: nil,
		},
		{
			name:        "Ошибка: Сессия Remote, ключа нет на сервере (ErrKeyEmpty)",
			sessionType: sharedUsecases.SessionTypeRemote,
			mockBehavior: func(remote *domainMocks.MockEncryptedKeyRepository, local *domainMocks.MockEncryptedKeyRepository, pp *mocks.MockPasswdProvider) {
				local.EXPECT().GetEncryptedKey(ctx).Return(nil, domain.ErrKeyNotFound).Times(1)
				remote.EXPECT().GetEncryptedKey(ctx).Return(nil, domain.ErrKeyEmpty).Times(1)
			},
			wantErr: domain.ErrKeyEmpty,
		},
		{
			name:        "Фоллбэк: Сессия Remote, сервер недоступен, но данные успешно прочитаны из кэша",
			sessionType: sharedUsecases.SessionTypeRemote,
			mockBehavior: func(remote *domainMocks.MockEncryptedKeyRepository, local *domainMocks.MockEncryptedKeyRepository, pp *mocks.MockPasswdProvider) {
				local.EXPECT().GetEncryptedKey(ctx).Return(localKeyBytes, nil).Times(1)
				remote.EXPECT().GetEncryptedKey(ctx).Return(nil, errors.New("network timeout")).Times(1)
				pp.EXPECT().GetPassword(ctx).Return(testPassword, nil).Times(1)
			},
			wantErr: nil,
		},
		{
			name:        "Ошибка: Сессия Remote, сервер упал и локальный кэш пуст",
			sessionType: sharedUsecases.SessionTypeRemote,
			mockBehavior: func(remote *domainMocks.MockEncryptedKeyRepository, local *domainMocks.MockEncryptedKeyRepository, pp *mocks.MockPasswdProvider) {
				local.EXPECT().GetEncryptedKey(ctx).Return(nil, domain.ErrKeyNotFound).Times(1)
				remote.EXPECT().GetEncryptedKey(ctx).Return(nil, errors.New("network timeout")).Times(1)
			},
			wantErr: errors.New("сервер недоступен и локальный кэш пуст"),
		},
		{
			name:        "Ошибка: Сбой ввода пароля в процессе дешифрования",
			sessionType: sharedUsecases.SessionTypeLocal,
			mockBehavior: func(remote *domainMocks.MockEncryptedKeyRepository, local *domainMocks.MockEncryptedKeyRepository, pp *mocks.MockPasswdProvider) {
				local.EXPECT().GetEncryptedKey(ctx).Return(localKeyBytes, nil).Times(1)
				pp.EXPECT().GetPassword(ctx).Return("", errors.New("terminal Abort")).Times(1)
			},
			wantErr: errors.New("ошибка ввода пароля"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRemote := domainMocks.NewMockEncryptedKeyRepository(ctrl)
			mockLocal := domainMocks.NewMockEncryptedKeyRepository(ctrl)
			mockPP := mocks.NewMockPasswdProvider(ctrl)

			tt.mockBehavior(mockRemote, mockLocal, mockPP)

			ks := domain.NewKeyStorage(
				func(encryptedBlock []byte, password string) ([]byte, error) {
					if password != testPassword {
						t.Errorf("в KeyDecryptor передан неверный пароль: %s", password)
					}
					return []byte("decrypted_master_key"), nil
				},
				nil, nil, nil,
			)

			uc := usecases.NewGetUserKeyUseCase(mockRemote, mockLocal, tt.sessionType)

			err := uc.Execute(ctx, ks, mockPP)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("ожидалась ошибка %v, но получен nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) && !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("получена ошибка: %q, ожидалась: %q", err.Error(), tt.wantErr.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("не ожидалось ошибки, но получена: %v", err)
			}
		})
	}
}
