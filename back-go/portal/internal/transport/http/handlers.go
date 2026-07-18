package http

import (
	"encoding/json"
	"io"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/portal/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/portal/internal/service"
)

func parseBody(c *fiber.Ctx, out any) { _ = json.Unmarshal(c.Body(), out) }

func validationError(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "VALIDATION", "message": msg})
}

func parseIDs(raw []float64) []int64 {
	out := make([]int64, 0, len(raw))
	for _, v := range raw {
		out = append(out, int64(v))
	}
	return out
}

// ── Топики ───────────────────────────────────────────────────────

func (h *handlers) listTopics(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	resp, err := h.eps.ListTopics(c.Context(), endpoint.CompanyReq{CompanyID: companyID})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"topics": resp})
}

type topicBody struct {
	Name  string  `json:"name"`
	Color *string `json:"color"`
	Icon  *string `json:"icon"`
}

func (h *handlers) createTopic(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	var body topicBody
	parseBody(c, &body)
	resp, err := h.eps.CreateTopic(c.Context(), endpoint.WriteTopicReq{
		CompanyID: companyID, UserID: currentUser(c).ID,
		Name: body.Name, Color: body.Color, Icon: body.Icon,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) updateTopic(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	var body topicBody
	parseBody(c, &body)
	resp, err := h.eps.UpdateTopic(c.Context(), endpoint.WriteTopicReq{
		CompanyID: companyID, ID: pathID(c),
		Name: body.Name, Color: body.Color, Icon: body.Icon,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteTopic(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	if _, err := h.eps.DeleteTopic(c.Context(), endpoint.TopicReq{CompanyID: companyID, ID: pathID(c)}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

// ── Посты ────────────────────────────────────────────────────────

func boolQuery(c *fiber.Ctx, key string) *bool {
	v := c.Query(key)
	if v == "" {
		return nil
	}
	b := v == "true" || v == "1"
	return &b
}

func (h *handlers) listPosts(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	var topicID *int64
	if v := c.Query("topic_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			topicID = &n
		}
	}
	resp, err := h.eps.ListPosts(c.Context(), endpoint.ListPostsReq{
		CompanyID: companyID, ViewerID: currentUser(c).ID,
		Params: service.PostListParams{
			TopicID: topicID, Pinned: boolQuery(c, "pinned"), Search: c.Query("search"),
			Tag: c.Query("tag"), Limit: c.QueryInt("limit"), Cursor: c.Query("cursor"),
		},
	})
	if err != nil {
		return h.respondError(c, err)
	}
	// Форма страницы — {"pinned": [...], "posts": [...], "next_cursor": ...}
	// (см. service.PostFeed).
	return c.JSON(resp)
}

// popularTags — топ хештегов компании для панели «Популярные теги» ленты.
func (h *handlers) popularTags(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	resp, err := h.eps.PopularTags(c.Context(), endpoint.PopularTagsReq{
		CompanyID: companyID, Limit: c.QueryInt("limit"),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"tags": resp})
}

func (h *handlers) getPost(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	resp, err := h.eps.GetPost(c.Context(), endpoint.PostReq{
		CompanyID: companyID, ID: pathID(c), ViewerID: currentUser(c).ID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) markView(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	if _, err := h.eps.MarkView(c.Context(), endpoint.PostReq{
		CompanyID: companyID, ID: pathID(c), ViewerID: currentUser(c).ID,
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

type postBody struct {
	TopicID *int64  `json:"topic_id"`
	Title   *string `json:"title"`
	Body    string  `json:"body"`
}

func (h *handlers) createPost(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	var body postBody
	parseBody(c, &body)
	resp, err := h.eps.CreatePost(c.Context(), endpoint.WritePostReq{
		CompanyID: companyID, UserID: currentUser(c).ID,
		TopicID: body.TopicID, Title: body.Title, Body: body.Body,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) updatePost(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	u := currentUser(c)
	var body postBody
	parseBody(c, &body)
	resp, err := h.eps.UpdatePost(c.Context(), endpoint.WritePostReq{
		CompanyID: companyID, ID: pathID(c), UserID: u.ID, RoleLevel: u.RoleLevel,
		TopicID: body.TopicID, Title: body.Title, Body: body.Body,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deletePost(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	u := currentUser(c)
	if _, err := h.eps.DeletePost(c.Context(), endpoint.PinReq{
		CompanyID: companyID, ID: pathID(c), UserID: u.ID, RoleLevel: u.RoleLevel,
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (h *handlers) pinPost(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	u := currentUser(c)
	// Тело опционально: {"days": N} — автоистечение пина (null/0 — бессрочно).
	var body struct {
		Days *int `json:"days"`
	}
	parseBody(c, &body)
	days := 0
	if body.Days != nil {
		days = *body.Days
	}
	resp, err := h.eps.Pin(c.Context(), endpoint.PinReq{
		CompanyID: companyID, ID: pathID(c), UserID: u.ID, RoleLevel: u.RoleLevel, Days: days,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) unpinPost(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	u := currentUser(c)
	resp, err := h.eps.Unpin(c.Context(), endpoint.PinReq{
		CompanyID: companyID, ID: pathID(c), UserID: u.ID, RoleLevel: u.RoleLevel,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// ── Вложения ─────────────────────────────────────────────────────

func (h *handlers) upload(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "NO_FILE", "message": "Файл не передан"})
	}
	if fileHeader.Size > uploadMaxBytes {
		return validationError(c, "Файл слишком большой (макс. 25 МБ)")
	}
	f, err := fileHeader.Open()
	if err != nil {
		return h.respondError(c, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, uploadMaxBytes+1))
	if err != nil {
		return h.respondError(c, err)
	}
	if int64(len(data)) > uploadMaxBytes {
		return validationError(c, "Файл слишком большой (макс. 25 МБ)")
	}
	u := currentUser(c)
	resp, err := h.eps.Upload(c.Context(), endpoint.UploadReq{
		CompanyID: companyID, PostID: pathID(c), UserID: u.ID, RoleLevel: u.RoleLevel,
		FileName: fileHeader.Filename, Mime: fileHeader.Header.Get(fiber.HeaderContentType), Data: data,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) removeAttachment(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	u := currentUser(c)
	if _, err := h.eps.RemoveAttachment(c.Context(), endpoint.AttachmentReq{
		CompanyID: companyID, ID: pathID(c), UserID: u.ID, RoleLevel: u.RoleLevel,
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

// ── Комментарии ──────────────────────────────────────────────────

func (h *handlers) listComments(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	resp, err := h.eps.ListComments(c.Context(), endpoint.ListCommentsReq{
		CompanyID: companyID, PostID: pathID(c), ViewerID: currentUser(c).ID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"comments": resp})
}

func (h *handlers) createComment(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	var body struct {
		Text      string `json:"text"`
		ReplyToID *int64 `json:"reply_to_id"`
	}
	parseBody(c, &body)
	resp, err := h.eps.CreateComment(c.Context(), endpoint.CreateCommentReq{
		CompanyID: companyID, PostID: pathID(c), AuthorID: currentUser(c).ID,
		Text: body.Text, ReplyToID: body.ReplyToID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

// commentID — параметр :id в маршрутах /comments/:id (свой namespace, не
// вложен под /posts/:id).
func commentID(c *fiber.Ctx) int64 {
	id, _ := c.ParamsInt("id")
	return int64(id)
}

func (h *handlers) deleteComment(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	u := currentUser(c)
	if _, err := h.eps.DeleteComment(c.Context(), endpoint.DeleteCommentReq{
		CompanyID: companyID, CommentID: commentID(c), UserID: u.ID, RoleLevel: u.RoleLevel,
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

// likeComment — toggle лайка комментария (как реакции мессенджера: одна
// ручка на «поставить» и «снять»).
func (h *handlers) likeComment(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	resp, err := h.eps.LikeComment(c.Context(), endpoint.LikeCommentReq{
		CompanyID: companyID, CommentID: commentID(c), UserID: currentUser(c).ID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// ── Реакции ──────────────────────────────────────────────────────

type reactionBody struct {
	Emoji string `json:"emoji"`
}

func (h *handlers) addReaction(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	var body reactionBody
	parseBody(c, &body)
	if _, err := h.eps.AddReaction(c.Context(), endpoint.ReactionReq{
		CompanyID: companyID, PostID: pathID(c), UserID: currentUser(c).ID, Emoji: body.Emoji,
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"added": true})
}

func (h *handlers) removeReaction(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	emoji := strings.TrimSpace(c.Query("emoji"))
	if emoji == "" {
		var body reactionBody
		parseBody(c, &body)
		emoji = body.Emoji
	}
	if _, err := h.eps.RemoveReaction(c.Context(), endpoint.ReactionReq{
		CompanyID: companyID, PostID: pathID(c), UserID: currentUser(c).ID, Emoji: emoji,
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

// ── Непрочитанные (бейдж в навигации) ────────────────────────────

func (h *handlers) unreadCount(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	resp, err := h.eps.UnreadCount(c.Context(), endpoint.SeenReq{CompanyID: companyID, UserID: currentUser(c).ID})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"count": resp})
}

func (h *handlers) markSeen(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	if _, err := h.eps.MarkSeen(c.Context(), endpoint.SeenReq{CompanyID: companyID, UserID: currentUser(c).ID}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

// ── Пересылка в мессенджер ───────────────────────────────────────

func (h *handlers) forwardPost(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	var body struct {
		ConversationIDs []float64 `json:"conversation_ids"`
		UserIDs         []float64 `json:"user_ids"`
	}
	parseBody(c, &body)
	resp, err := h.eps.ForwardPost(c.Context(), endpoint.ForwardReq{
		CompanyID: companyID, PostID: pathID(c), SenderID: currentUser(c).ID,
		ConversationIDs: parseIDs(body.ConversationIDs), UserIDs: parseIDs(body.UserIDs),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}
