package domain

import "github.com/DmitriyODS/gw2/back-go/pkg/apierror"

// Error — общая бизнес-ошибка платформы (pkg/apierror): REST-ответ
// {"error": code, "message": ...} с HTTP-статусом.
type Error = apierror.Error

func NewError(code, message string, httpStatus int) *Error {
	return apierror.New(code, message, httpStatus)
}

// AsDomainError — достать *Error из цепочки; nil, если это не бизнес-ошибка.
func AsDomainError(err error) *Error { return apierror.As(err) }

var (
	ErrNotFound       = NewError("NOT_FOUND", "Напоминание не найдено", 404)
	ErrTitleRequired  = NewError("VALIDATION", "Укажите, о чём напомнить", 400)
	ErrTimeRequired   = NewError("VALIDATION", "Укажите дату и время", 400)
	ErrBadRepeat      = NewError("VALIDATION", "Неизвестный вид повтора", 400)
	ErrBadLink        = NewError("VALIDATION", "Неизвестная привязка напоминания", 400)
	ErrBadSnooze      = NewError("VALIDATION", "Отложить можно на 1–1440 минут", 400)
	ErrNothingToRepeat = NewError("VALIDATION", "У разового напоминания нет следующего срока", 400)
)
