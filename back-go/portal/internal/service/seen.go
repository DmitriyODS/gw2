package service

import "context"

// UnreadCount — число непросмотренных постов компании для бейджа в навигации:
// посты позже последней отметки просмотра (без отметки — все), кроме своих.
func (s *Service) UnreadCount(ctx context.Context, userID, companyID int64) (int, error) {
	seenAt, err := s.repo.SeenAt(ctx, userID, companyID)
	if err != nil {
		return 0, err
	}
	return s.repo.CountPostsAfter(ctx, companyID, userID, seenAt)
}

// MarkSeen — отметить портал просмотренным (заход в раздел). Событий в шину
// не публикует: бейдж — личное состояние, другим клиентам оно не нужно.
func (s *Service) MarkSeen(ctx context.Context, userID, companyID int64) error {
	return s.repo.MarkSeen(ctx, userID, companyID)
}
