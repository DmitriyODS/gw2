// Package filestore — запись загруженных в календари файлов/картинок в хранилище
// (pkg/storage: локальный том или S3). Ключ на диске случайный (без утечки
// исходного имени), оригинальное имя хранится в метаданных записи.
package filestore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"mime"
	"path/filepath"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/calendar/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/storage"
)

const subdir = "calendar"

type Store struct {
	st storage.Storage
}

var _ domain.FileStore = (*Store)(nil)

func New(st storage.Storage) *Store { return &Store{st: st} }

// Save — записать файл под случайным ключом с сохранением расширения исходного
// файла. Возвращает относительный путь calendar/<hex><ext>.
func (s *Store) Save(fileName string, data []byte) (string, error) {
	name := make([]byte, 16)
	if _, err := rand.Read(name); err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(filepath.Base(fileName)))
	if len(ext) > 16 {
		ext = "" // защита от мусорного «расширения»
	}
	key := subdir + "/" + hex.EncodeToString(name) + ext
	if err := s.st.Put(context.Background(), key, data, contentType(ext)); err != nil {
		return "", err
	}
	return key, nil
}

func (s *Store) Remove(paths []string) {
	s.st.Remove(context.Background(), paths...)
}

func contentType(ext string) string {
	if ct := mime.TypeByExtension(ext); ct != "" {
		return ct
	}
	return "application/octet-stream"
}
