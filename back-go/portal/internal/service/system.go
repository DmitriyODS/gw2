package service

import (
	"context"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
)

// systemPostDedupWindow — окно дедупликации системных постов: повторный вызов
// с тем же (company_id, system_kind, author_user_id) — это ретрай хука
// (fire-and-forget у вызывающего), а не второе событие.
const systemPostDedupWindow = 10 * time.Minute

// CreateSystemPost — системный пост от имени пользователя (gRPC
// portal.v1.PortalService, зовёт petsvc: celebrating-посты вида
// 'pet_evolved'). Компания должна быть активна; дедуп в 10-минутном окне
// идемпотентно возвращает уже созданный пост без нового события.
func (s *Service) CreateSystemPost(ctx context.Context, companyID, authorID int64, systemKind, title, body string) (*domain.Post, error) {
	systemKind = strings.TrimSpace(systemKind)
	if systemKind == "" {
		return nil, domain.ErrSystemKindReq
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, domain.ErrPostBodyReq
	}
	active, err := s.users.CompanyActive(ctx, &companyID)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, domain.ErrCompanyDisabled
	}
	prev, err := s.repo.FindRecentSystemPost(ctx, companyID, authorID, systemKind, time.Now().Add(-systemPostDedupWindow))
	if err != nil {
		return nil, err
	}
	if prev != nil {
		return prev, nil
	}
	p := &domain.Post{
		CompanyID: companyID, AuthorID: authorID,
		Title: normTitle(&title), Body: body, SystemKind: &systemKind,
	}
	if err := s.repo.CreatePost(ctx, p); err != nil {
		return nil, err
	}
	p.Attachments = []domain.Attachment{}
	p.ReactionCount = map[string]int{}
	p.MyReactions = []string{}
	s.bus.Publish(ctx, "post:new", []string{roomAll}, postPayload(p))
	return p, nil
}
