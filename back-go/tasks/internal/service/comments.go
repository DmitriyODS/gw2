package service

import (
	"context"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
)

// ensureCanEditComment — автор может всегда, остальные — MANAGER+. Роль актора
// приходит ИЗ ТОКЕНА (роль в активной компании): в users её больше нет, поэтому
// чтение из БД здесь всегда давало бы 0 и глухой 403 даже менеджеру.
func ensureCanEditComment(c *domain.Comment, userID int64, actorLevel int) error {
	if c.AuthorID == userID {
		return nil
	}
	if actorLevel < domain.LevelManager {
		return domain.NewError("FORBIDDEN", "Нет прав на действие", 403)
	}
	return nil
}

func (s *Service) ListComments(ctx context.Context, taskID int64, companyID *int64) ([]dto.Comment, error) {
	if _, err := s.taskInCompany(ctx, taskID, companyID); err != nil {
		return nil, err
	}
	comments, err := s.comments.ListComments(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return dto.NewComments(comments), nil
}

func (s *Service) CreateComment(ctx context.Context, taskID, authorID int64, companyID *int64, text string) (*dto.Comment, error) {
	if _, err := s.taskInCompany(ctx, taskID, companyID); err != nil {
		return nil, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, domain.NewError("EMPTY", "Пустой текст", 422)
	}
	comment := &domain.Comment{TaskID: taskID, AuthorID: authorID, Text: text}
	if err := s.comments.CreateComment(ctx, comment); err != nil {
		return nil, err
	}

	created, err := s.comments.GetComment(ctx, comment.ID)
	if err != nil {
		return nil, err
	}
	out := dto.NewComment(created)
	s.bus.Publish(ctx, "comment:new", []string{roomAll}, out)
	return &out, nil
}

func (s *Service) UpdateComment(ctx context.Context, commentID, userID int64, actorLevel int, companyID *int64, text string) (*dto.Comment, error) {
	comment, err := s.comments.GetComment(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if comment == nil || comment.DeletedAt != nil {
		return nil, domain.NewError("NOT_FOUND", "Комментарий не найден", 404)
	}
	// Комментарий чужой компании неотличим от несуществующего.
	if _, err := s.taskInCompany(ctx, comment.TaskID, companyID); err != nil {
		return nil, domain.NewError("NOT_FOUND", "Комментарий не найден", 404)
	}
	if err := ensureCanEditComment(comment, userID, actorLevel); err != nil {
		return nil, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, domain.NewError("EMPTY", "Пустой текст", 422)
	}
	if err := s.comments.UpdateCommentText(ctx, commentID, text, time.Now().UTC()); err != nil {
		return nil, err
	}

	updated, err := s.comments.GetComment(ctx, commentID)
	if err != nil {
		return nil, err
	}
	out := dto.NewComment(updated)
	s.bus.Publish(ctx, "comment:updated", []string{roomAll}, out)
	return &out, nil
}

func (s *Service) DeleteComment(ctx context.Context, taskID, commentID, userID int64, actorLevel int, companyID *int64) error {
	comment, err := s.comments.GetComment(ctx, commentID)
	if err != nil {
		return err
	}
	if comment == nil || comment.DeletedAt != nil {
		return domain.NewError("NOT_FOUND", "Комментарий не найден", 404)
	}
	if _, err := s.taskInCompany(ctx, comment.TaskID, companyID); err != nil {
		return domain.NewError("NOT_FOUND", "Комментарий не найден", 404)
	}
	if err := ensureCanEditComment(comment, userID, actorLevel); err != nil {
		return err
	}
	if err := s.comments.SoftDeleteComment(ctx, commentID, time.Now().UTC()); err != nil {
		return err
	}
	s.bus.Publish(ctx, "comment:deleted", []string{roomAll}, map[string]any{
		"task_id": taskID, "comment_id": commentID,
	})
	return nil
}
