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
	ErrNoteNotFound   = NewError("NOT_FOUND", "Заметка не найдена", 404)
	ErrFolderNotFound = NewError("NOT_FOUND", "Папка не найдена", 404)
	ErrTagNotFound    = NewError("NOT_FOUND", "Тег не найден", 404)
	ErrShareNotFound  = NewError("NOT_FOUND", "Ссылка не найдена или отозвана", 404)
	ErrReadOnly       = NewError("FORBIDDEN", "Ссылка даёт доступ только для чтения", 403)
	ErrRateLimited    = NewError("RATE_LIMITED", "Слишком много правок, попробуйте чуть позже", 429)
	ErrBadAccess      = NewError("VALIDATION", "Режим доступа: view или edit", 400)
	ErrBadColor       = NewError("VALIDATION", "Неизвестный цвет", 400)
	ErrNameRequired   = NewError("VALIDATION", "Укажите название", 400)
	ErrFolderCycle    = NewError("VALIDATION", "Нельзя переместить папку внутрь самой себя", 400)

	ErrMemberNotFound   = NewError("NOT_FOUND", "Пользователь не найден", 404)
	ErrSelfShare        = NewError("VALIDATION", "Нельзя поделиться с самим собой", 400)
	ErrMemberReadOnly   = NewError("FORBIDDEN", "Доступно только для чтения", 403)
	ErrBadCollabKind    = NewError("VALIDATION", "Тип collab-события: join, leave, cursor или doc", 400)
	ErrNotCompanyMember = NewError("FORBIDDEN", "Вы не состоите в этой компании", 403)
	ErrBadTarget        = NewError("VALIDATION", "Аудитория шаринга: user или company", 400)
	ErrNothingToExport  = NewError("EMPTY_EXPORT", "Нечего экспортировать — здесь пока нет заметок", 404)
)
