package domain

import (
	"errors"

	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
)

// Error — общая бизнес-ошибка платформы (pkg/apierror): REST-ответ
// {"error": code, "message": ...} (пустой message опускается — как
// jsonify({"error": "NOT_FOUND"})); в gRPC уезжает полем
// error {code, message, http_status}.
type Error = apierror.Error

func NewError(code, message string, httpStatus int) *Error {
	return apierror.New(code, message, httpStatus)
}

// AsDomainError — достать *Error из цепочки; nil, если это не бизнес-ошибка.
func AsDomainError(err error) *Error { return apierror.As(err) }

// ErrSecretMisconfigured — AI_KEY_ENCRYPTION_KEY не задан или некорректен
// (аналог AiSecretMisconfigured во Flask: сознательный hard-fail шифрования,
// молча хранить ключи открытым текстом недопустимо).
var ErrSecretMisconfigured = errors.New("AI_KEY_ENCRYPTION_KEY не задан или некорректен")
