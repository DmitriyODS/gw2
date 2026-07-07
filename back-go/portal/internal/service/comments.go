package service

import (
	"context"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
)

func (s *Service) ListComments(ctx context.Context, companyID, postID int64) ([]*domain.Comment, error) {
	if _, err := s.requirePost(ctx, companyID, postID); err != nil {
		return nil, err
	}
	return s.repo.ListComments(ctx, postID)
}

func (s *Service) CreateComment(ctx context.Context, companyID, postID, authorID int64, text string) (*domain.Comment, error) {
	if _, err := s.requirePost(ctx, companyID, postID); err != nil {
		return nil, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, domain.ErrCommentTextReq
	}
	c := &domain.Comment{PostID: postID, AuthorID: authorID, Text: text}
	if err := s.repo.CreateComment(ctx, c); err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, "comment:new", []string{roomAll}, commentPayload(c))
	return c, nil
}

// DeleteComment — автор комментария или администратор компании. Пост
// (и его компания) определяется по самому комментарию — REST-маршрут
// DELETE /comments/:id не вложен под пост.
func (s *Service) DeleteComment(ctx context.Context, companyID, commentID, userID int64, roleLevel int) error {
	c, err := s.repo.GetComment(ctx, commentID)
	if err != nil {
		return err
	}
	if c == nil {
		return domain.ErrCommentNotFound
	}
	if _, err := s.requirePost(ctx, companyID, c.PostID); err != nil {
		return domain.ErrCommentNotFound
	}
	if c.AuthorID != userID && roleLevel < domain.LevelAdmin {
		return domain.ErrForbidden
	}
	if err := s.repo.DeleteComment(ctx, commentID); err != nil {
		return err
	}
	s.bus.Publish(ctx, "comment:deleted", []string{roomAll}, map[string]any{
		"id": commentID, "post_id": c.PostID, "company_id": companyID,
	})
	return nil
}

func commentPayload(c *domain.Comment) map[string]any {
	return map[string]any{
		"id": c.ID, "post_id": c.PostID, "author_id": c.AuthorID,
		"text": c.Text, "created_at": c.CreatedAt,
	}
}
