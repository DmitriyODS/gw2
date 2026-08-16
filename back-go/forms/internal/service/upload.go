package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/forms/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/chunkupload"
	"github.com/DmitriyODS/gw2/back-go/pkg/records"
)

/* Файлы, приложенные к ответам.

   Мелкое приезжает одним запросом, всё крупнее chunkupload.Threshold — частями
   (общий движок платформы). Место тратится из квоты ВЛАДЕЛЬЦА формы: отвечает
   часто гость, у которого своей квоты нет вовсе. */

// Upload — файл участника, отвечающего на форму.
func (s *Service) Upload(ctx context.Context, userID, formID, questionID int64, fileName, mime string, data []byte) (*domain.UploadedFile, error) {
	form, err := s.uploadTarget(ctx, userID, formID)
	if err != nil {
		return nil, err
	}
	if err := s.checkQuestionSize(ctx, formID, questionID, int64(len(data))); err != nil {
		return nil, err
	}
	return s.saveUpload(ctx, form, fileName, mime, data)
}

// uploadTarget — форма, в которую разрешено грузить: право отвечать плюс
// открытый приём (в закрытую форму файлы не нужны).
func (s *Service) uploadTarget(ctx context.Context, userID, formID int64) (*domain.Form, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	form, err := s.require(ctx, a, formID, domain.AccessRespond)
	if err != nil {
		return nil, err
	}
	if err := s.acceptable(ctx, form); err != nil {
		return nil, err
	}
	return form, nil
}

// checkQuestionSize — потолок размера задаёт сам вопрос (составитель формы
// решает, сколько весит приложение). Неизвестный вопрос проверку не проходит
// молча: остаётся общий потолок платформы.
func (s *Service) checkQuestionSize(ctx context.Context, formID, questionID, size int64) error {
	if size > domain.MaxFileSize {
		return domain.ErrFileTooBig
	}
	if questionID == 0 {
		return nil
	}
	q, err := s.repo.GetQuestion(ctx, formID, questionID)
	if err != nil || q == nil {
		return err
	}
	if _, limit := q.FileLimits(); size > limit {
		return domain.ErrFileTooBig
	}
	return nil
}

// saveUpload — записать файл и вернуть метаданные (их кладут в значение
// файлового вопроса). Картинки получают миниатюру: таблица ответов показывает
// превью, и без неё страница тянула бы оригиналы.
func (s *Service) saveUpload(ctx context.Context, form *domain.Form, fileName, mime string, data []byte) (*domain.UploadedFile, error) {
	if len(data) == 0 {
		return nil, domain.ErrEmptyFile
	}
	if len(data) > domain.MaxFileSize {
		return nil, domain.ErrFileTooBig
	}
	quotaUser, companyID := quotaScope(form)
	path, err := s.files.SaveFor(ctx, quotaUser, companyID, fileName, data)
	if err != nil {
		return nil, err
	}
	out := &domain.UploadedFile{Path: path, Name: fileName, Mime: mime, Size: int64(len(data))}
	if isImage(mime, fileName) {
		if thumb, opaque := records.Thumbnail(data, records.ThumbMax); thumb != nil {
			// Неудача миниатюры оригинал не отменяет: показывать будет он сам.
			if tp, err := s.files.SaveFor(ctx, quotaUser, companyID, "thumb"+records.ThumbExt(opaque), thumb); err == nil {
				out.Thumb = tp
			}
		}
	}
	return out, nil
}

// ── Чанковая загрузка ──

// BeginUpload — завести загрузку по частям. Права и потолок размера проверяем
// ДО первого байта: незачем принимать гигабайт, чтобы отказать на сборке.
func (s *Service) BeginUpload(ctx context.Context, userID, formID, questionID int64, fileName, mime string, size int64) (chunkupload.Session, error) {
	form, err := s.uploadTarget(ctx, userID, formID)
	if err != nil {
		return chunkupload.Session{}, err
	}
	if size <= 0 {
		return chunkupload.Session{}, domain.ErrEmptyFile
	}
	if err := s.checkQuestionSize(ctx, formID, questionID, size); err != nil {
		return chunkupload.Session{}, err
	}
	quotaUser, companyID := quotaScope(form)
	return s.uploads.Init(ctx, chunkupload.Session{
		UserID:    userID,
		CompanyID: companyID,
		Scope:     strconv.FormatInt(formID, 10),
		FileName:  fileName,
		Mime:      mime,
		TotalSize: size,
		// Владельца квоты запоминаем в сессии: собирать файл будем позже, и
		// форму к тому моменту придётся искать заново.
		QuotaUserID: quotaUser,
	})
}

func (s *Service) WriteChunk(ctx context.Context, userID int64, code string, index int, data []byte) (chunkupload.Session, error) {
	return s.uploads.Chunk(ctx, code, userID, index, data)
}

// FinishUpload — собрать файл из принятых частей. Миниатюру здесь не делаем:
// такие размеры — это файлы, а не картинки анкеты.
func (s *Service) FinishUpload(ctx context.Context, userID int64, code string) (*domain.UploadedFile, error) {
	sess, err := s.uploads.Get(ctx, code, userID)
	if err != nil {
		return nil, err
	}
	if !sess.Complete() {
		return nil, domain.ErrEmptyFile
	}
	reader := s.uploads.Reader(ctx, sess)
	defer reader.Close()

	path, err := s.files.SaveStreamFor(ctx, sess.QuotaUserID, sess.CompanyID,
		sess.FileName, reader, sess.TotalSize)
	if err != nil {
		return nil, err
	}
	s.uploads.Done(ctx, sess)
	return &domain.UploadedFile{
		Path: path, Name: sess.FileName, Mime: sess.Mime, Size: sess.TotalSize,
	}, nil
}

func (s *Service) CancelUpload(ctx context.Context, userID int64, code string) error {
	return s.uploads.Cancel(ctx, code, userID)
}

func isImage(mime, fileName string) bool {
	if strings.HasPrefix(strings.ToLower(mime), "image/") {
		return true
	}
	// Браузер не всегда присылает тип (перетаскивание из архива) — судим по имени.
	name := strings.ToLower(fileName)
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg"} {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}
