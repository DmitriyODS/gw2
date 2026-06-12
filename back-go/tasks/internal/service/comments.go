package service

import (
	"context"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
)

// ensureCanEditComment — автор может всегда, остальные — MANAGER+.
func (s *Service) ensureCanEditComment(ctx context.Context, c *domain.Comment, userID int64) error {
	if c.AuthorID == userID {
		return nil
	}
	user, err := s.users.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil || user.RoleLevel < domain.LevelManager {
		return domain.NewError("FORBIDDEN", "Нет прав на действие", 403)
	}
	return nil
}

func (s *Service) ListComments(ctx context.Context, taskID int64) ([]dto.Comment, error) {
	task, err := s.tasks.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, domain.NewError("TASK_NOT_FOUND", "Задача не найдена", 404)
	}
	comments, err := s.comments.ListComments(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return dto.NewComments(comments), nil
}

func (s *Service) CreateComment(ctx context.Context, taskID, authorID int64, text string) (*dto.Comment, error) {
	task, err := s.tasks.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, domain.NewError("TASK_NOT_FOUND", "Задача не найдена", 404)
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

func (s *Service) UpdateComment(ctx context.Context, commentID, userID int64, text string) (*dto.Comment, error) {
	comment, err := s.comments.GetComment(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if comment == nil || comment.DeletedAt != nil {
		return nil, domain.NewError("NOT_FOUND", "Комментарий не найден", 404)
	}
	if err := s.ensureCanEditComment(ctx, comment, userID); err != nil {
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

func (s *Service) DeleteComment(ctx context.Context, taskID, commentID, userID int64) error {
	comment, err := s.comments.GetComment(ctx, commentID)
	if err != nil {
		return err
	}
	if comment == nil || comment.DeletedAt != nil {
		return domain.NewError("NOT_FOUND", "Комментарий не найден", 404)
	}
	if err := s.ensureCanEditComment(ctx, comment, userID); err != nil {
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
