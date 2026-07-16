package service

import (
	"context"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
)

func (s *Service) ListComments(ctx context.Context, companyID, postID, viewerID int64) ([]*domain.Comment, error) {
	if _, err := s.requirePost(ctx, companyID, postID); err != nil {
		return nil, err
	}
	return s.repo.ListComments(ctx, postID, viewerID)
}

// CreateComment — комментарий или ответ на другой комментарий (replyToID).
// Родитель обязан жить в ТОМ ЖЕ посте: иначе ответом можно было бы утащить
// ветку в чужое обсуждение.
func (s *Service) CreateComment(ctx context.Context, companyID, postID, authorID int64,
	text string, replyToID *int64) (*domain.Comment, error) {

	if _, err := s.requirePost(ctx, companyID, postID); err != nil {
		return nil, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, domain.ErrCommentTextReq
	}
	if replyToID != nil {
		parent, err := s.repo.GetComment(ctx, *replyToID)
		if err != nil {
			return nil, err
		}
		if parent == nil || parent.PostID != postID {
			return nil, domain.ErrCommentNotFound
		}
	}
	c := &domain.Comment{PostID: postID, AuthorID: authorID, Text: text, ReplyToID: replyToID}
	if err := s.repo.CreateComment(ctx, c); err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, "comment:new", []string{roomAll}, commentPayload(c, companyID))
	return c, nil
}

// DeleteComment — автор комментария или администратор компании. Пост
// (и его компания) определяется по самому комментарию — REST-маршрут
// DELETE /comments/:id не вложен под пост. Ветка ответов уходит вместе с
// родителем (каскад FK) — клиент перечитывает обсуждение по событию.
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

// LikeComment — toggle лайка (как реакции мессенджера): своё состояние и
// счётчик возвращаются вызывающему, остальным приезжает событие.
func (s *Service) LikeComment(ctx context.Context, companyID, commentID, userID int64) (*domain.Comment, error) {
	c, err := s.repo.GetComment(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, domain.ErrCommentNotFound
	}
	if _, err := s.requirePost(ctx, companyID, c.PostID); err != nil {
		return nil, domain.ErrCommentNotFound
	}
	liked, count, err := s.repo.ToggleCommentLike(ctx, commentID, userID)
	if err != nil {
		return nil, err
	}
	c.Liked, c.LikeCount = liked, count
	s.bus.Publish(ctx, "comment:liked", []string{roomAll}, map[string]any{
		"id": c.ID, "post_id": c.PostID, "company_id": companyID,
		"like_count": count,
	})
	return c, nil
}

func commentPayload(c *domain.Comment, companyID int64) map[string]any {
	return map[string]any{
		"id": c.ID, "post_id": c.PostID, "author_id": c.AuthorID,
		"reply_to_id": c.ReplyToID, "text": c.Text, "created_at": c.CreatedAt,
		"like_count": c.LikeCount, "company_id": companyID,
	}
}
