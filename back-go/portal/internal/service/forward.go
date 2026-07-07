package service

import (
	"context"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
)

// ForwardResult — итог пересылки: сколько диалогов получили плашку поста и
// сколько получателей отвалилось (не найден диалог/нет доступа и т.п.).
type ForwardResult struct {
	Forwarded int `json:"forwarded"`
	Failed    int `json:"failed"`
}

// ForwardPost — переслать пост как структурную плашку kind='post' в один или
// несколько диалогов мессенджера. user_ids резолвятся в диалог через
// EnsureDialog (диалога может ещё не быть — форвард первым сообщением
// перепискe допустим, как и в остальном мессенджере), conversation_ids
// используются как есть (уже открытый диалог отправителя; участие
// отправителя проверяет msgsvc). Дубликаты диалогов схлопываются, любая
// ошибка ветки считается в failed. Событие message:new публикует сам msgsvc
// (gw2:messenger:events) — так его видит и pushsvc.
func (s *Service) ForwardPost(ctx context.Context, companyID, postID, senderID int64, conversationIDs, userIDs []int64) (ForwardResult, error) {
	if s.messenger == nil {
		return ForwardResult{}, domain.NewError("MESSENGER_UNAVAILABLE", "Мессенджер недоступен", 503)
	}
	p, err := s.requirePost(ctx, companyID, postID)
	if err != nil {
		return ForwardResult{}, err
	}
	preview := s.postPreview(ctx, p)

	var res ForwardResult
	seen := map[int64]bool{}
	convIDs := make([]int64, 0, len(conversationIDs)+len(userIDs))
	add := func(id int64) {
		if !seen[id] {
			seen[id] = true
			convIDs = append(convIDs, id)
		}
	}
	for _, cid := range conversationIDs {
		add(cid)
	}
	for _, uid := range userIDs {
		convID, err := s.messenger.EnsureDialog(ctx, senderID, uid)
		if err != nil {
			s.log.Warn("portal.forward_ensure_dialog_failed", "user_id", uid, "error", err)
			res.Failed++
			continue
		}
		add(convID)
	}

	for _, convID := range convIDs {
		if _, _, err := s.messenger.CreatePostMessage(ctx, convID, senderID, postID, preview); err != nil {
			s.log.Warn("portal.forward_failed", "post_id", postID, "conversation_id", convID, "error", err)
			res.Failed++
			continue
		}
		res.Forwarded++
	}
	return res, nil
}

const excerptLen = 150

// postPreview — снапшот поста для плашки в мессенджере: заголовок (или
// первые слова тела, если без заголовка), сокращённый текст, обложка —
// первое вложение-картинка.
func (s *Service) postPreview(ctx context.Context, p *domain.Post) domain.PostPreview {
	title := p.Body
	if p.Title != nil && *p.Title != "" {
		title = *p.Title
	}
	title = truncateRunes(strings.TrimSpace(title), excerptLen)

	excerpt := truncateRunes(strings.TrimSpace(p.Body), excerptLen)

	var cover string
	atts, err := s.repo.ListAttachments(ctx, p.ID)
	if err != nil {
		s.log.Warn("portal.forward_attachments_failed", "post_id", p.ID, "error", err)
	}
	for _, a := range atts {
		if a.Mime != nil && strings.HasPrefix(*a.Mime, "image/") {
			cover = "/uploads/" + a.FilePath
			break
		}
	}
	return domain.PostPreview{Title: title, Excerpt: excerpt, CoverURL: cover}
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
