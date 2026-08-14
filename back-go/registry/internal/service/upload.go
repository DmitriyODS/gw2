package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/pkg/chunkupload"
	"github.com/DmitriyODS/gw2/back-go/pkg/records"
	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

/* Загрузка файлов полей записи.

   Мелкое приезжает одним запросом, всё крупнее chunkupload.Threshold — частями
   (общий движок платформы). Порог решает КЛИЕНТ, но и одиночный путь проверяет
   размер: мимо интерфейса тоже ходят. */

// Upload — файл, загруженный участником с правом вести записи.
func (s *Service) Upload(ctx context.Context, userID, registryID int64, fileName, mime string, data []byte) (*domain.UploadedFile, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	reg, err := s.require(ctx, a, registryID, domain.AccessEdit)
	if err != nil {
		return nil, err
	}
	return s.saveUpload(ctx, reg, fileName, mime, data)
}

// saveUpload — записать файл/картинку и вернуть метаданные (их кладут в
// значение поля типа image/file соответствующей записи).
//
// Для картинок рядом сохраняется миниатюра: таблица записей показывает превью,
// и без неё страница из тридцати строк тянула бы тридцать оригиналов.
func (s *Service) saveUpload(ctx context.Context, reg *domain.Registry, fileName, mime string, data []byte) (*domain.UploadedFile, error) {
	if len(data) == 0 {
		return nil, domain.ErrEmptyFile
	}
	image := isImage(mime, fileName)
	if image && len(data) > domain.MaxImageSize {
		return nil, domain.ErrImageTooBig
	}
	if len(data) > domain.MaxFileSize {
		return nil, domain.ErrFileTooBig
	}

	userID, companyID := quotaScope(reg)
	path, err := s.files.SaveFor(ctx, userID, companyID, fileName, data)
	if err != nil {
		return nil, err
	}
	out := &domain.UploadedFile{
		Path: path, Name: fileName, Mime: mime, Size: int64(len(data)),
	}
	if image {
		if thumb, opaque := records.Thumbnail(data, records.ThumbMax); thumb != nil {
			// Неудача миниатюры оригинал не отменяет: показывать будет он сам.
			if tp, err := s.files.SaveFor(ctx, userID, companyID, "thumb"+records.ThumbExt(opaque), thumb); err == nil {
				out.Thumb = tp
			}
		}
	}
	return out, nil
}

// ── Чанковая загрузка ──

// BeginUpload — завести загрузку по частям. Права и потолок размера проверяем
// ДО первого байта: незачем принимать гигабайт, чтобы отказать на сборке.
func (s *Service) BeginUpload(ctx context.Context, userID, registryID int64, fileName, mime string, size int64) (chunkupload.Session, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return chunkupload.Session{}, err
	}
	reg, err := s.require(ctx, a, registryID, domain.AccessEdit)
	if err != nil {
		return chunkupload.Session{}, err
	}
	if size <= 0 {
		return chunkupload.Session{}, domain.ErrEmptyFile
	}
	if size > domain.MaxFileSize {
		return chunkupload.Session{}, domain.ErrFileTooBig
	}
	if isImage(mime, fileName) && size > domain.MaxImageSize {
		return chunkupload.Session{}, domain.ErrImageTooBig
	}
	quotaUser, companyID := quotaScope(reg)
	return s.uploads.Init(ctx, chunkupload.Session{
		UserID:    userID,
		CompanyID: companyID,
		Scope:     strconv.FormatInt(registryID, 10),
		FileName:  fileName,
		Mime:      mime,
		TotalSize: size,
		// Владельца квоты запоминаем в сессии: собирать файл будем позже, и
		// реестр к тому моменту придётся искать заново.
		QuotaUserID: quotaUser,
	})
}

// WriteChunk — принять часть загрузки.
func (s *Service) WriteChunk(ctx context.Context, userID int64, code string, index int, data []byte) (chunkupload.Session, error) {
	return s.uploads.Chunk(ctx, code, userID, index, data)
}

// FinishUpload — собрать файл из принятых частей. Миниатюру здесь не делаем:
// такие размеры — это файлы, а не обложки (обложка не переваливает за 2 МБ и
// едет одним запросом).
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
	// Браузер не всегда присылает тип (drag-and-drop из архива) — судим по имени.
	name := strings.ToLower(fileName)
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg"} {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}
