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
	ErrFormNotFound     = NewError("NOT_FOUND", "Форма не найдена", 404)
	ErrResponseNotFound = NewError("NOT_FOUND", "Ответ не найден", 404)
	ErrSectionNotFound  = NewError("VALIDATION", "Раздел формы не найден", 400)

	// Права. Существование чужой формы не раскрываем — на чтение отвечаем 404,
	// нехватку уровня различаем только тому, кто форму уже видит.
	ErrForbidden  = NewError("FORBIDDEN", "Недостаточно прав для этого действия", 403)
	ErrOwnerOnly  = NewError("FORBIDDEN", "Это может сделать только владелец формы", 403)
	ErrShareSelf  = NewError("VALIDATION", "Форма и так ваша", 400)
	ErrNoAudience = NewError("VALIDATION", "Не выбрано, с кем поделиться", 400)

	// Приём ответов.
	ErrNotOpen = NewError("FORM_CLOSED",
		"Форма не принимает ответы", 409)
	ErrNotStarted = NewError("FORM_NOT_STARTED",
		"Форма ещё не открыта для ответов", 409)
	ErrClosed = NewError("FORM_CLOSED",
		"Приём ответов завершён", 409)
	ErrLimitReached = NewError("FORM_LIMIT",
		"Форма набрала максимальное число ответов", 409)
	ErrAlreadyAnswered = NewError("ALREADY_ANSWERED",
		"Вы уже отвечали на эту форму", 409)
	ErrAuthRequired = NewError("AUTH_REQUIRED",
		"Чтобы ответить на эту форму, войдите в аккаунт", 401)
	ErrEditNotAllowed = NewError("EDIT_NOT_ALLOWED",
		"Изменять отправленный ответ нельзя", 403)
	ErrEmailRequired = NewError("VALIDATION",
		"Укажите адрес почты", 400)
	ErrNoQuestions = NewError("VALIDATION",
		"В форме нет ни одного вопроса", 400)
	// Мест не осталось: последнее занял тот, кто отправил ответ первым.
	ErrNoSlots = NewError("NO_SLOTS",
		"Свободных мест по выбранному варианту не осталось", 409)

	// Файлы.
	ErrFileTooBig  = NewError("FILE_TOO_BIG", "Файл больше допустимого размера", 413)
	ErrImageTooBig = NewError("IMAGE_TOO_BIG", "Картинка больше допустимого размера", 413)
	ErrEmptyFile   = NewError("EMPTY_FILE", "Пустой файл", 400)
)

// ErrRequired — не заполнен обязательный вопрос.
func ErrRequired(title string) *Error {
	return NewError("VALIDATION", "Ответьте на обязательный вопрос «"+title+"»", 400)
}
