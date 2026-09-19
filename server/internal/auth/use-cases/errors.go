package usecases

import "errors"

var ErrWrongLoginOrPassword = errors.New("неверный логин или пароль")
var ErrLoginBusy = errors.New("данный логин занят")
