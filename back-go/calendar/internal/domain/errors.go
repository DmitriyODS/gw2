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
	ErrCalendarNotFound = NewError("NOT_FOUND", "Календарь не найден", 404)
	ErrEntryNotFound    = NewError("NOT_FOUND", "Запись не найдена", 404)
	ErrNoCompany        = NewError("BAD_REQUEST", "Нет активной компании", 400)
	ErrEventAtRequired  = NewError("VALIDATION", "Укажите дату и время записи", 400)
)
