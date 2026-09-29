package service

import (
	"context"
	"io"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/pkg/records"
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
	return s.registerAttachment(ctx, companyID, postID, path, s.saveThumb(ctx, userID, companyID, mime, data),
		fileName, mime, int64(len(data)))
}

// feedThumbMax — сторона миниатюры ленты. Одиночная картинка идёт во всю
// ширину карточки, и табличные 320 px (records.ThumbMax) там были бы мылом;
// 1080 — чётко на телефоне и всё равно в разы легче оригинала.
const feedThumbMax = 1080

// saveThumb — миниатюра картинки для ленты: без неё каждый пост тянул на
// телефон оригинал в мегабайты. Не вышло (не картинка, мала, экзотический
// формат, не хватило места) — лента покажет оригинал, вложение не страдает.
// Файлам, пришедшим частями, миниатюру не строим: декодировать картинку
// крупнее порога значит держать в памяти сотни мегабайт пикселей.
func (s *Service) saveThumb(ctx context.Context, userID, companyID int64, mime string, data []byte) *string {
	if !strings.HasPrefix(mime, "image/") {
		return nil
	}
	thumb, opaque := records.Thumbnail(data, feedThumbMax)
	if thumb == nil {
		return nil
	}
	path, err := s.files.SaveFor(ctx, userID, companyID, "thumb"+records.ThumbExt(opaque), thumb)
	if err != nil {
		return nil
	}
	return &path
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
	return s.registerAttachment(ctx, companyID, postID, path, nil, fileName, mime, size)
}

// registerAttachment — завести запись о уже сохранённом объекте.
func (s *Service) registerAttachment(ctx context.Context, companyID, postID int64,
	path string, thumb *string, fileName, mime string, size int64) (*domain.Attachment, error) {

	a := &domain.Attachment{
		PostID: postID, FilePath: path, ThumbPath: thumb, Name: fileName,
		Size: size, Mime: nonEmpty(mime),
	}
	if err := s.repo.AddAttachment(ctx, a); err != nil {
		return nil, err
	}
	a.SetURLs()
	s.bus.Publish(ctx, "post:updated", companyRoom(companyID), map[string]any{
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
	s.files.RemoveFor(ctx, userID, companyID, a.Paths())
	s.bus.Publish(ctx, "post:updated", companyRoom(companyID), map[string]any{
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
