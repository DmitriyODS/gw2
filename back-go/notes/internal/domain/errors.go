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
	ErrNoteNotFound  = NewError("NOT_FOUND", "Заметка не найдена", 404)
	ErrGroupNotFound = NewError("NOT_FOUND", "Группа не найдена", 404)
	ErrShareNotFound = NewError("NOT_FOUND", "Ссылка не найдена или отозвана", 404)
	ErrReadOnly      = NewError("FORBIDDEN", "Ссылка даёт доступ только для чтения", 403)
	ErrRateLimited   = NewError("RATE_LIMITED", "Слишком много правок, попробуйте чуть позже", 429)
	ErrBadAccess     = NewError("VALIDATION", "Режим доступа: view или edit", 400)
	ErrBadColor      = NewError("VALIDATION", "Неизвестный цвет заметки", 400)
	ErrNameRequired  = NewError("VALIDATION", "Укажите название группы", 400)
)
