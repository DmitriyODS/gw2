package domain

import "github.com/DmitriyODS/gw2/back-go/pkg/apierror"

// Error — общая бизнес-ошибка платформы (pkg/apierror): REST-ответ
// {"error": code, "message": ...} с её HTTP-статусом; в gRPC уезжает
// полем error {code, message, http_status}.
type Error = apierror.Error

func NewError(code, message string, httpStatus int) *Error {
	return apierror.New(code, message, httpStatus)
}

// AsDomainError — достать *Error из цепочки; nil, если это не бизнес-ошибка.
func AsDomainError(err error) *Error { return apierror.As(err) }

// ErrSoldOut — тираж лимитированного товара распродан. Возвращают и сервис
// (превентивная проверка витрины), и репозиторий (авторитетная проверка в
// одной транзакции с INSERT покупки — см. ShopRepo.RecordPurchase).
var ErrSoldOut = NewError("SOLD_OUT", "Тираж этого товара распродан", 422)

// ErrPetAway — питомец в приключении: платные действия (кормление, прогулка,
// лечение, поглаживание чужого) недоступны, пока он не вернулся.
var ErrPetAway = NewError("PET_AWAY", "Питомец в приключении", 422)

// ErrAdventureLimit — дневной лимит стартов приключений исчерпан.
var ErrAdventureLimit = NewError("ADVENTURE_LIMIT", "Приключений на сегодня достаточно", 429)

// ErrPetOnVacation — хозяин в отпуске: питомец тоже отдыхает, действия
// (свои и поглаживания коллег) недоступны, показатели заморожены.
var ErrPetOnVacation = NewError("PET_ON_VACATION", "Питомец в отпуске вместе с хозяином", 422)
