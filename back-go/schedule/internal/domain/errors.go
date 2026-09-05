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
	// Существование чужого расписания не раскрываем: нет доступа — 404.
	ErrScheduleNotFound = NewError("NOT_FOUND", "Расписание не найдено", 404)
	ErrItemNotFound     = NewError("NOT_FOUND", "Занятие не найдено", 404)
	ErrCategoryNotFound = NewError("NOT_FOUND", "Категория не найдена", 404)
	ErrShareNotFound    = NewError("NOT_FOUND", "Ссылка не найдена или отозвана", 404)

	// Шаринг у расписания только на чтение: вести занятия может владелец.
	ErrReadOnly   = NewError("FORBIDDEN", "Расписание доступно только для чтения", 403)
	ErrShareSelf  = NewError("VALIDATION", "Расписание и так ваше", 400)
	ErrNoAudience = NewError("VALIDATION", "Не выбрано, с кем поделиться", 400)

	ErrNameRequired  = NewError("VALIDATION", "Укажите название расписания", 400)
	ErrTitleRequired = NewError("VALIDATION", "Укажите название занятия", 400)
	ErrBadWeekday    = NewError("VALIDATION", "Укажите день недели", 400)
	ErrBadTime       = NewError("VALIDATION", "Конец занятия должен быть позже начала", 400)
	ErrBadRepeat     = NewError("VALIDATION",
		"Задайте либо недели цикла, либо своё правило повтора", 400)
	ErrBadFieldType = NewError("VALIDATION", "Неизвестный тип поля", 400)
	ErrBadImport    = NewError("VALIDATION", "Файл не похож на расписание", 400)
)
