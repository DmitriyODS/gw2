package service

import (
	"context"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
)

func (s *Service) ListTopics(ctx context.Context, companyID int64) ([]*domain.Topic, error) {
	return s.repo.ListTopics(ctx, companyID)
}

// CreateTopic — новый тематический раздел (только администратор компании,
// проверка роли — в HTTP-транспорте через RequireRole).
func (s *Service) CreateTopic(ctx context.Context, companyID, userID int64, name string, color, icon *string) (*domain.Topic, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrTopicNameReq
	}
	t := &domain.Topic{CompanyID: companyID, Name: name, Color: color, Icon: icon, CreatedBy: userID}
	if err := s.repo.CreateTopic(ctx, t); err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, "topic:created", []string{roomAll}, topicPayload(t))
	return t, nil
}

func (s *Service) UpdateTopic(ctx context.Context, companyID, id int64, name string, color, icon *string) (*domain.Topic, error) {
	t, err := s.requireTopic(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrTopicNameReq
	}
	if err := s.repo.UpdateTopic(ctx, id, name, color, icon); err != nil {
		return nil, err
	}
	t.Name, t.Color, t.Icon = name, color, icon
	s.bus.Publish(ctx, "topic:updated", []string{roomAll}, topicPayload(t))
	return t, nil
}

func (s *Service) DeleteTopic(ctx context.Context, companyID, id int64) error {
	if _, err := s.requireTopic(ctx, companyID, id); err != nil {
		return err
	}
	if err := s.repo.DeleteTopic(ctx, id); err != nil {
		return err
	}
	s.bus.Publish(ctx, "topic:deleted", []string{roomAll}, map[string]any{
		"id": id, "company_id": companyID,
	})
	return nil
}

func topicPayload(t *domain.Topic) map[string]any {
	return map[string]any{
		"id": t.ID, "company_id": t.CompanyID, "name": t.Name,
		"color": t.Color, "icon": t.Icon, "created_by": t.CreatedBy, "created_at": t.CreatedAt,
	}
}
