package service

import (
	"context"
	"io"

	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
)

// AddAttachment — сохранить файл и привязать его к посту (автор или
// администратор — та же проверка, что на правку поста).
func (s *Service) AddAttachment(ctx context.Context, companyID, postID, userID int64, roleLevel int, fileName, mime string, data []byte) (*domain.Attachment, error) {
	p, err := s.requirePost(ctx, companyID, postID)
	if err != nil {
		return nil, err
	}
	if !canManage(p, userID, roleLevel) {
		return nil, domain.ErrForbidden
	}
	path, err := s.files.SaveFor(ctx, userID, companyID, fileName, data)
	if err != nil {
		return nil, err
	}
	return s.registerAttachment(ctx, companyID, postID, path, fileName, mime, int64(len(data)))
}

/*
CheckAttachment — можно ли прикладывать к этому посту. Зовётся ДО первой

	части: отказывать на сборке поздно.
*/
func (s *Service) CheckAttachment(ctx context.Context, companyID, postID, userID int64, roleLevel int) error {
	p, err := s.requirePost(ctx, companyID, postID)
	if err != nil {
		return err
	}
	if !canManage(p, userID, roleLevel) {
		return domain.ErrForbidden
	}
	return nil
}

// AddAttachmentStream — вложение поста, собранное из частей.
func (s *Service) AddAttachmentStream(ctx context.Context, companyID, postID, userID int64,
	roleLevel int, fileName, mime string, size int64, r io.Reader) (*domain.Attachment, error) {

	if err := s.CheckAttachment(ctx, companyID, postID, userID, roleLevel); err != nil {
		return nil, err
	}
	path, err := s.files.SaveStreamFor(ctx, userID, companyID, fileName, r, size)
	if err != nil {
		return nil, err
	}
	return s.registerAttachment(ctx, companyID, postID, path, fileName, mime, size)
}

// registerAttachment — завести запись о уже сохранённом объекте.
func (s *Service) registerAttachment(ctx context.Context, companyID, postID int64,
	path, fileName, mime string, size int64) (*domain.Attachment, error) {

	a := &domain.Attachment{
		PostID: postID, FilePath: path, Name: fileName,
		Size: size, Mime: nonEmpty(mime),
	}
	if err := s.repo.AddAttachment(ctx, a); err != nil {
		return nil, err
	}
	a.URL = "/uploads/" + a.FilePath
	s.bus.Publish(ctx, "post:updated", []string{roomAll}, map[string]any{
		"id": postID, "company_id": companyID, "attachment_added": true,
	})
	return a, nil
}

// RemoveAttachment — удалить вложение поста (автор или администратор — та же
// проверка, что на добавление). Скоуп компании — через пост вложения.
func (s *Service) RemoveAttachment(ctx context.Context, companyID, attachmentID, userID int64, roleLevel int) error {
	a, err := s.repo.GetAttachment(ctx, attachmentID)
	if err != nil {
		return err
	}
	if a == nil {
		return domain.ErrAttachmentNotFound
	}
	p, err := s.requirePost(ctx, companyID, a.PostID)
	if err != nil {
		return err
	}
	if !canManage(p, userID, roleLevel) {
		return domain.ErrForbidden
	}
	if err := s.repo.DeleteAttachment(ctx, attachmentID); err != nil {
		return err
	}
	s.files.RemoveFor(ctx, userID, companyID, []string{a.FilePath})
	s.bus.Publish(ctx, "post:updated", []string{roomAll}, map[string]any{
		"id": a.PostID, "company_id": companyID, "attachment_removed": true,
	})
	return nil
}

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
