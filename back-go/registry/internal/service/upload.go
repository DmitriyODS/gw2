package service

import "github.com/DmitriyODS/gw2/back-go/registry/internal/domain"

// SaveUpload — записать загруженный файл/картинку и вернуть его метаданные
// (их кладут в значение поля типа image/file соответствующей записи).
func (s *Service) SaveUpload(fileName, mime string, data []byte) (*domain.UploadedFile, error) {
	path, err := s.files.Save(fileName, data)
	if err != nil {
		return nil, err
	}
	return &domain.UploadedFile{
		Path: path, Name: fileName, Mime: mime, Size: int64(len(data)),
	}, nil
}
