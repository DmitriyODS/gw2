package service

import (
	"context"
	"io"

	"github.com/DmitriyODS/gw2/back-go/calendar/internal/domain"
)

// uploadScope — чья квота платит за загружаемый файл. Файл грузится раньше
// записи, но уже для конкретного календаря: за календарь команды платит её
// создатель, за личный — хозяин. Без календаря файл ложится на загрузившего.
func (s *Service) uploadScope(ctx context.Context, userID, calendarID int64) (int64, int64, error) {
	if calendarID == 0 {
		return userID, 0, nil
	}
	cal, err := s.requireCalendar(ctx, userID, calendarID, domain.AccessEdit)
	if err != nil {
		return 0, 0, err
	}
	quotaUser, companyID := quotaScope(cal)
	return quotaUser, companyID, nil
}

// SaveUpload — записать загруженный файл/картинку и вернуть его метаданные
// (их кладут в значение поля типа image/file соответствующей записи).
func (s *Service) SaveUpload(ctx context.Context, userID, calendarID int64, fileName, mime string, data []byte) (*domain.UploadedFile, error) {
	quotaUser, companyID, err := s.uploadScope(ctx, userID, calendarID)
	if err != nil {
		return nil, err
	}
	path, err := s.files.SaveFor(ctx, quotaUser, companyID, fileName, data)
	if err != nil {
		return nil, err
	}
	return &domain.UploadedFile{
		Path: path, Name: fileName, Mime: mime, Size: int64(len(data)),
	}, nil
}

// UploadScope — плательщик для загрузки частями: проверяется при начале
// загрузки, чтобы чужой календарь не принял ни одного байта.
func (s *Service) UploadScope(ctx context.Context, userID, calendarID int64) (int64, int64, error) {
	return s.uploadScope(ctx, userID, calendarID)
}

// SaveUploadStream — то же для файла, пришедшего ЧАСТЯМИ: содержимое приезжает
// потоком, а размер известен заранее (его подтвердили принятые части).
// Плательщик уже проверен при начале загрузки.
func (s *Service) SaveUploadStream(ctx context.Context, quotaUser, companyID int64,
	fileName, mime string, size int64, r io.Reader) (*domain.UploadedFile, error) {

	path, err := s.files.SaveStreamFor(ctx, quotaUser, companyID, fileName, r, size)
	if err != nil {
		return nil, err
	}
	return &domain.UploadedFile{Path: path, Name: fileName, Mime: mime, Size: size}, nil
}
