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
	ErrRegistryNotFound = NewError("NOT_FOUND", "Реестр не найден", 404)
	ErrRecordNotFound   = NewError("NOT_FOUND", "Запись не найдена", 404)
	ErrNoCompany        = NewError("BAD_REQUEST", "Нет активной компании", 400)
	// Подразделы строятся только по списковому полю самого реестра: у остальных
	// типов нет заранее известного набора значений, из которого выйдут вкладки.
	ErrSectionFieldInvalid = NewError("VALIDATION",
		"Подразделы строятся только по списковому полю этого реестра", 400)

	// Права. Существование чужого реестра не раскрываем — на чтение отвечаем
	// 404, отказ в правке различаем только тому, кто реестр уже видит.
	ErrForbidden  = NewError("FORBIDDEN", "Недостаточно прав для этого действия", 403)
	ErrOwnerOnly  = NewError("FORBIDDEN", "Это может сделать только владелец реестра", 403)
	ErrShareSelf  = NewError("VALIDATION", "Реестр и так ваш", 400)
	ErrNoAudience = NewError("VALIDATION", "Не выбрано, с кем поделиться", 400)

	// Учётный реестр.
	ErrNotAccounting = NewError("VALIDATION", "Реестр не ведёт учёт выдач", 400)
	ErrAlreadyIssued = NewError("ALREADY_ISSUED", "Позиция уже выдана", 409)
	ErrNotIssued     = NewError("NOT_ISSUED", "Позиция сейчас на месте", 409)

	// Файлы.
	ErrFileTooBig  = NewError("FILE_TOO_BIG", "Файл больше допустимого размера", 413)
	ErrImageTooBig = NewError("IMAGE_TOO_BIG", "Обложка больше допустимого размера", 413)
	ErrEmptyFile   = NewError("EMPTY_FILE", "Пустой файл", 400)
)
