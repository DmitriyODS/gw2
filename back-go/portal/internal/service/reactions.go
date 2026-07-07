package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
)

// validEmoji — реакция: непустая и похожа на эмодзи, а не на произвольный
// текст (≤16 байт и ≤4 рун покрывают составные эмодзи с VS16/скин-тоном).
func validEmoji(emoji string) (string, error) {
	emoji = strings.TrimSpace(emoji)
	if emoji == "" {
		return "", domain.ErrEmojiRequired
	}
	if len(emoji) > 16 || utf8.RuneCountInString(emoji) > 4 {
		return "", domain.ErrEmojiInvalid
	}
	return emoji, nil
}

func (s *Service) AddReaction(ctx context.Context, companyID, postID, userID int64, emoji string) error {
	if _, err := s.requirePost(ctx, companyID, postID); err != nil {
		return err
	}
	emoji, err := validEmoji(emoji)
	if err != nil {
		return err
	}
	if err := s.repo.AddReaction(ctx, &domain.Reaction{PostID: postID, UserID: userID, Emoji: emoji}); err != nil {
		return err
	}
	s.bus.Publish(ctx, "reaction:added", []string{roomAll}, map[string]any{
		"post_id": postID, "user_id": userID, "emoji": emoji, "company_id": companyID,
	})
	return nil
}

func (s *Service) RemoveReaction(ctx context.Context, companyID, postID, userID int64, emoji string) error {
	if _, err := s.requirePost(ctx, companyID, postID); err != nil {
		return err
	}
	emoji, err := validEmoji(emoji)
	if err != nil {
		return err
	}
	if err := s.repo.RemoveReaction(ctx, postID, userID, emoji); err != nil {
		return err
	}
	s.bus.Publish(ctx, "reaction:removed", []string{roomAll}, map[string]any{
		"post_id": postID, "user_id": userID, "emoji": emoji, "company_id": companyID,
	})
	return nil
}
