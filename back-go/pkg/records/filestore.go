package records

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"mime"
	"path/filepath"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/pkg/storage"
)

// FileStore — запись загруженных файлов/картинок записей в хранилище
// (pkg/storage: локальный том или S3). Ключ на диске случайный (без утечки
// исходного имени), оригинальное имя хранится в метаданных записи. subdir —
// префикс сервиса ("registry"/"calendar").
type FileStore struct {
	st     storage.Storage
	subdir string
}

func NewFileStore(st storage.Storage, subdir string) *FileStore {
	return &FileStore{st: st, subdir: subdir}
}

// Save — записать файл под случайным ключом с сохранением расширения исходного
// файла. Возвращает относительный путь <subdir>/<hex><ext>.
func (s *FileStore) Save(fileName string, data []byte) (string, error) {
	name := make([]byte, 16)
	if _, err := rand.Read(name); err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(filepath.Base(fileName)))
	if len(ext) > 16 {
		ext = "" // защита от мусорного «расширения»
	}
	key := s.subdir + "/" + hex.EncodeToString(name) + ext
	if err := s.st.Put(context.Background(), key, data, contentType(ext)); err != nil {
		return "", err
	}
	return key, nil
}

func (s *FileStore) Remove(paths []string) {
	s.st.Remove(context.Background(), paths...)
}

func contentType(ext string) string {
	if ct := mime.TypeByExtension(ext); ct != "" {
		return ct
	}
	return "application/octet-stream"
}
