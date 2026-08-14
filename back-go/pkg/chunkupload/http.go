package chunkupload

import (
	"io"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
)

/* Готовые ручки приёма частей.

   Механика у всех разделов одна (завести сессию → слать части → собрать файл),
   различаются только две вещи: можно ли этому человеку сюда грузить и во что
   превращается собранный файл. Их сервис и передаёт — остальное общее, чтобы
   шесть разделов не разошлись в поведении докачки. */

// InitRequest — что просит клиент при заведении загрузки. Scope — контекст
// раздела (id реестра, папки, переписки); его разбирает сам сервис.
type InitRequest struct {
	FileName string `json:"file_name"`
	Mime     string `json:"mime"`
	Size     int64  `json:"size"`
	Scope    string `json:"scope"`
}

// Handlers — обвязка приёма частей для одного раздела.
type Handlers struct {
	Manager *Manager
	// UserID — кто грузит (у всех сервисов свой способ достать его из контекста).
	UserID func(c *fiber.Ctx) int64
	// Begin — проверить право на загрузку и дополнить сессию: чья квота платит
	// (CompanyID/QuotaUserID) и к чему относится файл (Scope). Отказ — обычная
	// доменная ошибка.
	Begin func(c *fiber.Ctx, in InitRequest, s *Session) error
	// Finish — собрать файл из потока в сущность раздела и вернуть то, что
	// раздел отдаёт клиенту.
	Finish func(c *fiber.Ctx, s Session, r io.Reader) (any, error)
	// Log — для ответов об ошибках.
	Respond func(c *fiber.Ctx, err error) error
}

// Mount — повесить ручки на префикс раздела: <prefix>/init, <prefix>/:code/chunk,
// <prefix>/:code/finish и отмену <prefix>/:code.
func (h Handlers) Mount(router fiber.Router, prefix string) {
	router.Post(prefix+"/init", h.init)
	router.Post(prefix+"/:code/chunk", h.chunk)
	router.Post(prefix+"/:code/finish", h.finish)
	router.Delete(prefix+"/:code", h.cancel)
}

func (h Handlers) init(c *fiber.Ctx) error {
	var in InitRequest
	if err := c.BodyParser(&in); err != nil {
		return h.fail(c, apierror.New("VALIDATION", "Непонятный запрос загрузки", 400))
	}
	if in.FileName == "" {
		return h.fail(c, apierror.New("VALIDATION", "Не указано имя файла", 400))
	}
	if in.Size <= 0 {
		return h.fail(c, apierror.New("EMPTY_FILE", "Пустой файл", 400))
	}

	s := Session{
		UserID:    h.UserID(c),
		Scope:     in.Scope,
		FileName:  in.FileName,
		Mime:      in.Mime,
		TotalSize: in.Size,
	}
	if err := h.Begin(c, in, &s); err != nil {
		return h.fail(c, err)
	}
	out, err := h.Manager.Init(c.Context(), s)
	if err != nil {
		return h.fail(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

// chunk — часть приходит СЫРЫМ телом, а не формой: обёртка multipart на каждый
// кусок — лишние проценты трафика и лишняя сборка в памяти.
func (h Handlers) chunk(c *fiber.Ctx) error {
	index := c.QueryInt("index", -1)
	if index < 0 {
		return h.fail(c, apierror.New("VALIDATION", "Не указан номер части", 400))
	}
	s, err := h.Manager.Chunk(c.Context(), c.Params("code"), h.UserID(c), index, c.Body())
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(s)
}

func (h Handlers) finish(c *fiber.Ctx) error {
	s, err := h.Manager.Get(c.Context(), c.Params("code"), h.UserID(c))
	if err != nil {
		return h.fail(c, err)
	}
	if !s.Complete() {
		return h.fail(c, apierror.New("UPLOAD_INCOMPLETE", "Файл дошёл не полностью", 400))
	}
	reader := h.Manager.Reader(c.Context(), s)
	defer reader.Close()

	out, err := h.Finish(c, s, reader)
	if err != nil {
		return h.fail(c, err)
	}
	// Части убираем только после успешной сборки: иначе неудача на записи
	// оставила бы человека без файла и без возможности повторить finish.
	h.Manager.Done(c.Context(), s)
	return c.Status(fiber.StatusCreated).JSON(out)
}

func (h Handlers) cancel(c *fiber.Ctx) error {
	if err := h.Manager.Cancel(c.Context(), c.Params("code"), h.UserID(c)); err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"cancelled": true})
}

func (h Handlers) fail(c *fiber.Ctx, err error) error {
	return h.Respond(c, err)
}
