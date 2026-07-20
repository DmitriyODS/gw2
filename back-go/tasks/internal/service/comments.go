package service

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
)

// mentionRe — @логин в тексте комментария: буквы/цифры/точка/подчёркивание
// (логины транслитом вида «ivanov.i.i»). @ должен стоять в начале или после
// несловного символа — иначе адреса вида «foo@bar» дали бы ложное упоминание
// (RE2 без lookbehind, поэтому границу ловим отдельной группой). Точки по
// краям обрезаются отдельно.
var mentionRe = regexp.MustCompile(`(^|[^\p{L}\p{N}_.@])@([\p{L}\p{N}_.]+)`)

// parseMentionLogins — уникальные нормализованные (lower) логины из @упоминаний.
func parseMentionLogins(text string) []string {
	matches := mentionRe.FindAllStringSubmatch(text, -1)
	seen := map[string]bool{}
	out := []string{}
	for _, m := range matches {
		login := strings.ToLower(strings.Trim(m[2], "."))
		if login == "" || seen[login] {
			continue
		}
		seen[login] = true
		out = append(out, login)
	}
	return out
}

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

func (s *Service) ListComments(ctx context.Context, taskID, userID int64, companyID *int64) (*dto.CommentList, error) {
	if _, err := s.taskInCompany(ctx, taskID, companyID); err != nil {
		return nil, err
	}
	comments, err := s.comments.ListComments(ctx, taskID)
	if err != nil {
		return nil, err
	}
	newCount, err := s.comments.CountNewComments(ctx, taskID, userID)
	if err != nil {
		return nil, err
	}
	return &dto.CommentList{Items: dto.NewComments(comments), NewCount: newCount}, nil
}

// MarkCommentsSeen — отметить комментарии задачи прочитанными для пользователя
// (гасит бейдж новых). Задача должна быть в активной компании сессии.
func (s *Service) MarkCommentsSeen(ctx context.Context, taskID, userID int64, companyID *int64) error {
	if _, err := s.taskInCompany(ctx, taskID, companyID); err != nil {
		return err
	}
	if err := s.comments.MarkCommentsSeen(ctx, taskID, userID); err != nil {
		return err
	}
	// Прочтение комментариев гасит и упоминания пользователя в этой задаче.
	return s.comments.MarkMentionsSeen(ctx, taskID, userID)
}

// notifyMentions — записать упоминания @логинов (члены компании, кроме автора)
// и разослать событие task:mention адресно каждому — чтобы у него на карточке
// задачи появился бейдж. Best-effort: ошибки только в лог, комментарий не роняют.
func (s *Service) notifyMentions(ctx context.Context, taskID, commentID, authorID int64, companyID *int64, text string) {
	if companyID == nil {
		return
	}
	logins := parseMentionLogins(text)
	if len(logins) == 0 {
		return
	}
	resolved, err := s.comments.ResolveMentions(ctx, *companyID, logins)
	if err != nil {
		s.log.Warn("mentions.resolve", "err", err)
		return
	}
	userIDs := make([]int64, 0, len(resolved))
	for _, uid := range resolved {
		if uid != authorID { // себя не уведомляем
			userIDs = append(userIDs, uid)
		}
	}
	if len(userIDs) == 0 {
		return
	}
	if err := s.comments.CreateMentions(ctx, taskID, commentID, userIDs); err != nil {
		s.log.Warn("mentions.create", "err", err)
		return
	}
	for _, uid := range userIDs {
		s.bus.Publish(ctx, "task:mention", []string{userRoom(uid)}, map[string]any{
			"task_id": taskID, "comment_id": commentID,
		})
	}
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
	s.notifyMentions(ctx, taskID, comment.ID, authorID, companyID, text)
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
