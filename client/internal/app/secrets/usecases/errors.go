package usecases

import "errors"

var (
	ErrIncorectSecretVersion = errors.New("некорректная версия секрета")
	ErrNotFound              = errors.New("секрет не найден")
	ErrServerUnavailable     = errors.New("ошибка направления запроса на сервер")
	ErrDataFromLocalCache    = errors.New("ошибка получения данных от сервера, данные получены из локального кэша")
)
