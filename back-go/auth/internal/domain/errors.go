package domain

import "github.com/DmitriyODS/gw2/back-go/pkg/apierror"

// Error — бизнес-ошибка с HTTP-статусом и кодом для фронта. АЛИАС на общий
// pkg/apierror.Error: единый формат ответов {"error": code, "message": ...}
// (+Extra: retry_after_sec, company_name) и, главное, распознаётся
// apierror.Respond/As в транспорте. Без алиаса (отдельным типом) любая
// бизнес-ошибка не распозналась бы и улетала бы 500 INTERNAL_ERROR.
type Error = apierror.Error

func NewError(code, message string, httpStatus int) *Error {
	return apierror.New(code, message, httpStatus)
}

func NewErrorExtra(code, message string, httpStatus int, extra map[string]any) *Error {
	return apierror.NewExtra(code, message, httpStatus, extra)
}

// AsDomainError — распаковать *Error из цепочки ошибок (nil, если это
// внутренняя ошибка, а не бизнес-ответ).
func AsDomainError(err error) *Error {
	return apierror.As(err)
}
