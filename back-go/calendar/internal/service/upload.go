package service

import (
	"context"
	"io"

	"github.com/DmitriyODS/gw2/back-go/calendar/internal/domain"
)

// SaveUpload — записать загруженный файл/картинку и вернуть его метаданные
// (их кладут в значение поля типа image/file соответствующей записи).
func (s *Service) SaveUpload(ctx context.Context, companyID, userID int64, fileName, mime string, data []byte) (*domain.UploadedFile, error) {
	path, err := s.files.SaveFor(ctx, userID, companyID, fileName, data)
	if err != nil {
		return nil, err
	}
	return &domain.UploadedFile{
		Path: path, Name: fileName, Mime: mime, Size: int64(len(data)),
	}, nil
}

// SaveUploadStream — то же для файла, пришедшего ЧАСТЯМИ: содержимое приезжает
// потоком, а размер известен заранее (его подтвердили принятые части).
func (s *Service) SaveUploadStream(ctx context.Context, companyID, userID int64,
	fileName, mime string, size int64, r io.Reader) (*domain.UploadedFile, error) {

	path, err := s.files.SaveStreamFor(ctx, userID, companyID, fileName, r, size)
	if err != nil {
		return nil, err
	}
	return &domain.UploadedFile{Path: path, Name: fileName, Mime: mime, Size: size}, nil
}
