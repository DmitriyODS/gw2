package domain

import "errors"

// Error — бизнес-ошибка домена. Коды стабильны: фронт показывает их в
// call:error, REST-клиенты получают {code, message} с http_status.
type Error struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func NewError(code, message string, httpStatus int) *Error {
	return &Error{Code: code, Message: message, HTTPStatus: httpStatus}
}

// AsDomainError — достать *Error из цепочки; nil, если это не бизнес-ошибка.
func AsDomainError(err error) *Error {
	var de *Error
	if errors.As(err, &de) {
		return de
	}
	return nil
}
