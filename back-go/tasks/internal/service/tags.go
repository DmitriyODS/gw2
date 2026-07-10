package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
)

// ── Теги задач ───────────────────────────────────────────────────
// Справочник компании (CRUD — менеджер, как отделы/этапы) + назначение
// набора тегов задаче (любой сотрудник). Теги общие для компании — в
// отличие от личного цвета карточки.

func (s *Service) ListTags(ctx context.Context, companyID int64) ([]dto.TagDTO, error) {
	items, err := s.tags.ListTags(ctx, companyID)
	if err != nil {
		return nil, err
	}
	return dto.NewTags(items), nil
}

func (s *Service) CreateTag(ctx context.Context, companyID int64, name, color string) (*dto.TagDTO, error) {
	existing, err := s.tags.GetTagByName(ctx, name, companyID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, domain.NewError("DUPLICATE", "Тег с таким именем уже существует", 409)
	}
	tag := &domain.Tag{Name: name, Color: color, CompanyID: companyID}
	if err := s.tags.CreateTag(ctx, tag); err != nil {
		return nil, err
	}
	out := dto.NewTag(tag)
	return &out, nil
}

func (s *Service) UpdateTag(ctx context.Context, companyID, tagID int64, name, color *string) (*dto.TagDTO, error) {
	tag, err := s.tags.GetTag(ctx, tagID)
	if err != nil {
		return nil, err
	}
	if tag == nil || tag.CompanyID != companyID {
		return nil, domain.NewError("NOT_FOUND", "Тег не найден", 404)
	}
	if name != nil && *name != tag.Name {
		existing, err := s.tags.GetTagByName(ctx, *name, companyID)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != tagID {
			return nil, domain.NewError("DUPLICATE", "Тег с таким именем уже существует", 409)
		}
	}
	fields := map[string]any{}
	if name != nil {
		fields["name"] = *name
		tag.Name = *name
	}
	if color != nil {
		fields["color"] = *color
		tag.Color = *color
	}
	if err := s.tags.UpdateTagFields(ctx, tagID, fields); err != nil {
		return nil, err
	}
	out := dto.NewTag(tag)
	return &out, nil
}

// DeleteTag — связи task_tags уходят каскадом FK; клиенты узнают о снятии
// тега с карточек при следующем fetch (событийно не рассылаем — редкое
// администраторское действие).
func (s *Service) DeleteTag(ctx context.Context, companyID, tagID int64) error {
	tag, err := s.tags.GetTag(ctx, tagID)
	if err != nil {
		return err
	}
	if tag == nil || tag.CompanyID != companyID {
		return domain.NewError("NOT_FOUND", "Тег не найден", 404)
	}
	return s.tags.DeleteTag(ctx, tagID)
}

// SetTaskTags — полная замена набора тегов задачи; все теги обязаны
// принадлежать активной компании. Рассылает task:updated (теги общие).
func (s *Service) SetTaskTags(ctx context.Context, taskID, userID int64,
	companyID *int64, tagIDs []int64) (*dto.Task, error) {

	task, err := s.taskInCompany(ctx, taskID, companyID)
	if err != nil {
		return nil, err
	}
	seen := map[int64]bool{}
	unique := make([]int64, 0, len(tagIDs))
	for _, id := range tagIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		tag, err := s.tags.GetTag(ctx, id)
		if err != nil {
			return nil, err
		}
		if tag == nil || tag.CompanyID != *companyID {
			return nil, domain.NewError("TAG_NOT_FOUND", "Тег не найден", 404)
		}
		unique = append(unique, id)
	}
	if err := s.tags.SetTaskTags(ctx, taskID, unique); err != nil {
		return nil, err
	}
	out, err := s.enrichTask(ctx, task, userID)
	if err != nil {
		return nil, err
	}
	s.broadcastTask(ctx, "task:updated", out)
	return &out, nil
}
