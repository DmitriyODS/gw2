package domain

import (
	"errors"

	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
)

// Error — общая бизнес-ошибка платформы (pkg/apierror): REST-ответ
// {"error": code, "message": ...} с её HTTP-статусом; в gRPC уезжает
// полем error {code, message, http_status}.
type Error = apierror.Error

func NewError(code, message string, httpStatus int) *Error {
	return apierror.New(code, message, httpStatus)
}

// AsDomainError — достать *Error из цепочки; nil, если это не бизнес-ошибка.
func AsDomainError(err error) *Error { return apierror.As(err) }

// ErrFolderNotFound — папки чатов нет или она чужая. Существование чужой не
// раскрываем: снаружи оба случая неотличимы.
var ErrFolderNotFound = NewError("FOLDER_NOT_FOUND", "Папка не найдена", 404)

// ErrDuplicateClientID — сообщение с этим ключом идемпотентности у
// отправителя уже есть (повтор отправки из очереди или второй вкладки).
var ErrDuplicateClientID = errors.New("message with this client_id already exists")
