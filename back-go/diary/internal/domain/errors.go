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
	ErrDiaryNotFound  = NewError("NOT_FOUND", "Ежедневник не найден", 404)
	ErrEntryNotFound  = NewError("NOT_FOUND", "Запись не найдена", 404)
	ErrReadOnly       = NewError("FORBIDDEN", "Ежедневник доступен только для чтения", 403)
	ErrDateRequired   = NewError("VALIDATION", "Укажите дату записи", 400)
	ErrTitleRequired  = NewError("VALIDATION", "Укажите название записи", 400)
	ErrShareNotFound  = NewError("NOT_FOUND", "Ссылка не найдена или отозвана", 404)
	ErrMemberNotFound = NewError("NOT_FOUND", "Пользователь не найден", 404)
	ErrSelfShare      = NewError("VALIDATION", "Нельзя поделиться с самим собой", 400)
	ErrOwnerOnly      = NewError("FORBIDDEN", "Это может сделать только владелец ежедневника", 403)
	ErrNotTeamMember  = NewError("NOT_TEAM_MEMBER", "Вы не состоите в этой команде", 403)
	// ErrMyDayFixed — «Мой день» служебный: его не переименовать, не удалить, не
	// раздать и не перенести в команду.
	ErrMyDayFixed = NewError("VALIDATION", "«Мой день» нельзя изменить", 400)
)
