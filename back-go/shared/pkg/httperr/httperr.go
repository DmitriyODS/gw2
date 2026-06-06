// Package httperr — общий формат HTTP-ошибок для всех сервисов.
//
// Каждый сервис в своём transport/http/errors.go держит switch по доменным
// sentinel-ошибкам и маппит их в Response через этот пакет. Универсальный
// маппер тут жить НЕ должен — он сделал бы сервисы связанными через общий
// dictionary ошибок.
package httperr

// Response — единый JSON-формат ошибки, который видит фронт.
type Response struct {
	Code    string       `json:"code"`              // машинно-читаемый: "invalid_credentials"
	Message string       `json:"message"`           // человеко-читаемый
	Fields  []FieldError `json:"fields,omitempty"`  // для validation_failed
}

// FieldError — описание одной валидационной ошибки.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// New создаёт Response без полей.
func New(code, message string) Response {
	return Response{Code: code, Message: message}
}

// Validation создаёт Response с полями.
func Validation(fields []FieldError) Response {
	return Response{
		Code:    "validation_failed",
		Message: "validation failed",
		Fields:  fields,
	}
}
