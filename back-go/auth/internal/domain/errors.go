package domain

import "errors"

// Error — бизнес-ошибка с HTTP-статусом и кодом для фронта. Формат ответа
// совпадает с прежними Flask-обработчиками: {"error": code, "message": ...}
// плюс дополнительные поля из Extra (retry_after_sec, company_name).
type Error struct {
	Code       string
	Message    string
	HTTPStatus int
	Extra      map[string]any
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func NewError(code, message string, httpStatus int) *Error {
	return &Error{Code: code, Message: message, HTTPStatus: httpStatus}
}

func NewErrorExtra(code, message string, httpStatus int, extra map[string]any) *Error {
	return &Error{Code: code, Message: message, HTTPStatus: httpStatus, Extra: extra}
}

// AsDomainError — распаковать *Error из цепочки ошибок (nil, если это
// внутренняя ошибка, а не бизнес-ответ).
func AsDomainError(err error) *Error {
	var de *Error
	if errors.As(err, &de) {
		return de
	}
	return nil
}
