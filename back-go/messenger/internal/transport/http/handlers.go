package http

import (
	"encoding/json"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

const (
	defaultMessagesLimit = 50
	maxMessagesLimit     = 200
	maxTextLength        = 10000
)

type handlers struct {
	eps endpoint.Endpoints
	log *slog.Logger
}

// respondError — бизнес-ошибка в форме {"error": code, "message": ...} с её
// HTTP-статусом; прочее — 500, как Flask-обработчик ошибок.
func (h *handlers) respondError(c *fiber.Ctx, err error) error {
	return apierror.Respond(c, err, h.log)
}

// currentUserID — id пользователя из Locals (после RequireAuth).
func currentUserID(c *fiber.Ctx) int64 {
	return pasetoauth.UserID(c)
}

// activeCompanyID — активная компания сессии из токена (authSource кладёт её в
// AuthInfo.User; в самих users её нет). Нужна для соло-чатов (pet/dev).
func activeCompanyID(c *fiber.Ctx) *int64 {
	if u, ok := pasetoauth.CurrentUser(c).(*domain.User); ok {
		return u.CompanyID
	}
	return nil
}

// validationError — форма marshmallow ValidationError: message — словарь
// {поле: [тексты]}, как возвращал Flask.
func validationError(c *fiber.Ctx, field, msg string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error":   "VALIDATION_ERROR",
		"message": fiber.Map{field: []string{msg}},
	})
}

// parseBody — request.get_json(silent=True) or {}: невалидный JSON не
// ошибка, просто пустое тело (required-поля отвалятся на валидации).
func parseBody(c *fiber.Ctx, out any) {
	_ = json.Unmarshal(c.Body(), out)
}

func pathID(c *fiber.Ctx) int64 {
	id, _ := c.ParamsInt("id")
	return int64(id)
}

// queryIntPtr — как request.args.get(type=int) во Flask: невалидное
// значение молча превращается в отсутствующее.
func queryIntPtr(c *fiber.Ctx, name string) *int64 {
	raw := c.Query(name)
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}

func scopeParam(c *fiber.Ctx) string {
	scope := strings.ToLower(c.Query("scope"))
	if scope == "" {
		scope = "me"
	}
	return scope
}

// ── Диалоги ──────────────────────────────────────────────────────

func (h *handlers) listConversations(c *fiber.Ctx) error {
	resp, err := h.eps.ListConversations(c.Context(), endpoint.ListConversationsRequest{
		UserID: currentUserID(c), CompanyID: activeCompanyID(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) openConversation(c *fiber.Ctx) error {
	var body struct {
		UserID *int64 `json:"user_id"`
	}
	parseBody(c, &body)
	if body.UserID == nil {
		return validationError(c, "user_id", "Missing data for required field.")
	}
	resp, err := h.eps.OpenConversation(c.Context(), endpoint.OpenConversationRequest{
		MeID: currentUserID(c), OtherUserID: *body.UserID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteConversation(c *fiber.Ctx) error {
	scope := scopeParam(c)
	resp, err := h.eps.DeleteConversation(c.Context(), endpoint.ScopedDeleteRequest{
		ID: pathID(c), UserID: currentUserID(c), Scope: scope,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true, "scope": scope, "physical": resp.(bool)})
}

func (h *handlers) toggleConversationPin(c *fiber.Ctx) error {
	resp, err := h.eps.ToggleConversationPin(c.Context(), endpoint.ConvUserRequest{
		ConversationID: pathID(c), UserID: currentUserID(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"is_pinned": resp.(bool)})
}

func (h *handlers) openDevChat(c *fiber.Ctx) error {
	resp, err := h.eps.OpenDevChat(c.Context(), endpoint.SoloChatRequest{
		UserID: currentUserID(c), CompanyID: activeCompanyID(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) openPetChat(c *fiber.Ctx) error {
	resp, err := h.eps.OpenPetChat(c.Context(), endpoint.SoloChatRequest{
		UserID: currentUserID(c), CompanyID: activeCompanyID(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) supportInbox(c *fiber.Ctx) error {
	resp, err := h.eps.SupportInbox(c.Context(), currentUserID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) unread(c *fiber.Ctx) error {
	resp, err := h.eps.TotalUnread(c.Context(), currentUserID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"total": resp.(int)})
}

// ── Сообщения ────────────────────────────────────────────────────

func (h *handlers) listMessages(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", defaultMessagesLimit)
	if limit > maxMessagesLimit {
		limit = maxMessagesLimit
	}
	resp, err := h.eps.ListMessages(c.Context(), endpoint.ListMessagesRequest{
		ConversationID: pathID(c),
		UserID:         currentUserID(c),
		BeforeID:       queryIntPtr(c, "before_id"),
		AfterID:        queryIntPtr(c, "after_id"),
		Limit:          limit,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) postMessage(c *fiber.Ctx) error {
	var body dto.MessageCreate
	parseBody(c, &body)
	if body.Text != nil && utf8.RuneCountInString(*body.Text) > maxTextLength {
		return validationError(c, "text", "Longer than maximum length 10000.")
	}
	resp, err := h.eps.SendMessage(c.Context(), endpoint.SendMessageRequest{
		ConversationID: pathID(c), SenderID: currentUserID(c), Body: body,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) forward(c *fiber.Ctx) error {
	var body dto.ForwardRequest
	parseBody(c, &body)
	if body.MessageID == nil {
		return validationError(c, "message_id", "Missing data for required field.")
	}
	resp, err := h.eps.ForwardMessage(c.Context(), endpoint.ForwardRequest{
		SenderID:        currentUserID(c),
		MessageID:       *body.MessageID,
		ConversationIDs: body.ConversationIDs,
		UserIDs:         body.UserIDs,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"forwarded": resp})
}

func (h *handlers) markRead(c *fiber.Ctx) error {
	resp, err := h.eps.MarkRead(c.Context(), endpoint.ConvUserRequest{
		ConversationID: pathID(c), UserID: currentUserID(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"updated": resp.(int)})
}

func (h *handlers) upload(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "NO_FILE", "message": "Файл не передан",
		})
	}
	f, err := fileHeader.Open()
	if err != nil {
		return h.respondError(c, err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return h.respondError(c, err)
	}
	resp, err := h.eps.UploadAttachment(c.Context(), endpoint.UploadRequest{
		UploaderID: currentUserID(c),
		FileName:   fileHeader.Filename,
		MimeType:   fileHeader.Header.Get(fiber.HeaderContentType),
		Data:       data,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) deleteMessage(c *fiber.Ctx) error {
	scope := scopeParam(c)
	resp, err := h.eps.DeleteMessage(c.Context(), endpoint.ScopedDeleteRequest{
		ID: pathID(c), UserID: currentUserID(c), Scope: scope,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true, "scope": scope, "for_all": resp.(bool)})
}

func (h *handlers) editMessage(c *fiber.Ctx) error {
	var body struct {
		Text *string `json:"text"`
	}
	parseBody(c, &body)
	if body.Text == nil {
		return validationError(c, "text", "Missing data for required field.")
	}
	if utf8.RuneCountInString(*body.Text) > maxTextLength {
		return validationError(c, "text", "Longer than maximum length 10000.")
	}
	resp, err := h.eps.EditMessage(c.Context(), endpoint.EditMessageRequest{
		MessageID: pathID(c), UserID: currentUserID(c), Text: *body.Text,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) toggleMessagePin(c *fiber.Ctx) error {
	resp, err := h.eps.ToggleMessagePin(c.Context(), endpoint.MsgUserRequest{
		MessageID: pathID(c), UserID: currentUserID(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	r := resp.(endpoint.MessagePinResponse)
	return c.JSON(fiber.Map{"pinned": r.Pinned, "message": r.Message})
}

func (h *handlers) listPinned(c *fiber.Ctx) error {
	resp, err := h.eps.ListPinnedMessages(c.Context(), endpoint.ConvUserRequest{
		ConversationID: pathID(c), UserID: currentUserID(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}
