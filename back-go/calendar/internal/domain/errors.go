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
	ErrEventAtRequired  = NewError("VALIDATION", "Укажите дату и время записи", 400)

	// Права. Существование чужого календаря не раскрываем — на чтение отвечаем
	// 404, отказ в правке различаем только тому, кто календарь уже видит.
	ErrForbidden     = NewError("FORBIDDEN", "Недостаточно прав для этого действия", 403)
	ErrOwnerOnly     = NewError("FORBIDDEN", "Это может сделать только владелец календаря", 403)
	ErrNotTeamMember = NewError("NOT_TEAM_MEMBER", "Вы не состоите в этой команде", 403)
	ErrMoveFailed    = NewError("MOVE_FAILED", "Не удалось перенести календарь, попробуйте ещё раз", 503)
)
