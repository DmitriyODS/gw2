// Package apierror — общая бизнес-ошибка платформы Groove Work.
//
// Единый формат для всех Go-микросервисов: REST-ответ
// {"error": CODE, "message": ...} (+произвольные Extra-поля вроде
// retry_after_sec) с HTTP-статусом ошибки; в gRPC уезжает полем
// error {code, message, http_status} (транспорт всегда OK).
package apierror

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

// Error — бизнес-ошибка с HTTP-статусом и кодом для фронта. Коды стабильны:
// фронт и REST-клиенты опираются на них, не на текст.
type Error struct {
	Code       string
	Message    string
	HTTPStatus int
	Extra      map[string]any
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func New(code, message string, httpStatus int) *Error {
	return &Error{Code: code, Message: message, HTTPStatus: httpStatus}
}

func NewExtra(code, message string, httpStatus int, extra map[string]any) *Error {
	return &Error{Code: code, Message: message, HTTPStatus: httpStatus, Extra: extra}
}

// As — распаковать *Error из цепочки ошибок (nil, если это внутренняя
// ошибка, а не бизнес-ответ).
func As(err error) *Error {
	var de *Error
	if errors.As(err, &de) {
		return de
	}
	return nil
}

// Respond — бизнес-ошибка в форме {"error": code, "message": ...}
// (+Extra-поля; пустой message опускается — как jsonify({"error": ...})
// во Flask) с её HTTP-статусом; прочее — 500, как Flask-обработчик ошибок.
func Respond(c *fiber.Ctx, err error, log *slog.Logger) error {
	if de := As(err); de != nil {
		body := fiber.Map{"error": de.Code}
		if de.Message != "" {
			body["message"] = de.Message
		}
		for k, v := range de.Extra {
			body[k] = v
		}
		return c.Status(de.HTTPStatus).JSON(body)
	}
	log.Error("http.internal_error", "path", c.Path(), "error", err)
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": "INTERNAL_ERROR", "message": "Внутренняя ошибка сервера",
	})
}
