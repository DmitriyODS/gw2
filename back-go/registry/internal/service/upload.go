package service

import (
	"context"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/pkg/records"
	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

// SaveUpload — записать загруженный файл/картинку и вернуть его метаданные
// (их кладут в значение поля типа image/file соответствующей записи).
//
// Для картинок рядом сохраняется миниатюра: таблица записей показывает превью,
// и без неё страница из тридцати строк тянула бы тридцать оригиналов.
func (s *Service) SaveUpload(ctx context.Context, companyID, userID int64, fileName, mime string, data []byte) (*domain.UploadedFile, error) {
	path, err := s.files.SaveFor(ctx, userID, companyID, fileName, data)
	if err != nil {
		return nil, err
	}
	out := &domain.UploadedFile{
		Path: path, Name: fileName, Mime: mime, Size: int64(len(data)),
	}
	if isImage(mime, fileName) {
		if thumb, opaque := records.Thumbnail(data, records.ThumbMax); thumb != nil {
			// Неудача миниатюры оригинал не отменяет: показывать будет он сам.
			if tp, err := s.files.SaveFor(ctx, userID, companyID, "thumb"+records.ThumbExt(opaque), thumb); err == nil {
				out.Thumb = tp
			}
		}
	}
	return out, nil
}

func isImage(mime, fileName string) bool {
	if strings.HasPrefix(strings.ToLower(mime), "image/") {
		return true
	}
	// Браузер не всегда присылает тип (drag-and-drop из архива) — судим по имени.
	name := strings.ToLower(fileName)
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif"} {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}
