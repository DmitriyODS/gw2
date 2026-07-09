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
	ErrTopicNotFound   = NewError("NOT_FOUND", "Раздел не найден", 404)
	ErrTopicNameReq    = NewError("VALIDATION", "Укажите название раздела", 400)
	ErrPostNotFound    = NewError("NOT_FOUND", "Пост не найден", 404)
	ErrPostBodyReq     = NewError("VALIDATION", "Укажите текст поста", 400)
	ErrCommentNotFound = NewError("NOT_FOUND", "Комментарий не найден", 404)
	ErrCommentTextReq  = NewError("VALIDATION", "Укажите текст комментария", 400)
	ErrEmojiRequired   = NewError("VALIDATION", "Укажите эмодзи реакции", 400)
	// EmojiInvalid — реакция длиннее, чем бывает эмодзи (≤16 байт / ≤4 рун):
	// в поле реакции пытаются протащить произвольный текст.
	ErrEmojiInvalid = NewError("VALIDATION", "Некорректная реакция", 422)
	ErrForbidden    = NewError("FORBIDDEN", "Недостаточно прав", 403)
	// BadCursor — нечитаемый keyset-курсор пагинации ленты (query ?cursor=).
	ErrBadCursor = NewError("VALIDATION", "Некорректный курсор пагинации", 400)
	// TooManyPinned — лимит одновременно закреплённых постов на компанию
	// (аналог SharePoint boost-лимита: не более 10, иначе закреплённые
	// перегружают ленту и теряют смысл «важного сверху»).
	ErrTooManyPinned = NewError("TOO_MANY_PINNED", "Уже закреплено 10 постов — открепите один, чтобы закрепить новый", 422)
)
