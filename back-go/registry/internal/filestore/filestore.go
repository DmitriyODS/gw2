// Package filestore — запись загруженных в реестры файлов/картинок в общий
// uploads-том (раздаёт nginx /uploads/). Имя на диске случайное (без утечки
// исходного имени в путь), оригинальное имя хранится в метаданных записи.
package filestore

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

const subdir = "registry"

type Store struct {
	root string
}

var _ domain.FileStore = (*Store)(nil)

func New(uploadFolder string) *Store { return &Store{root: uploadFolder} }

// Save — записать файл под случайным именем с сохранением расширения исходного
// файла. Возвращает относительный путь registry/<hex><ext>.
func (s *Store) Save(fileName string, data []byte) (string, error) {
	dir := filepath.Join(s.root, subdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := make([]byte, 16)
	if _, err := rand.Read(name); err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(filepath.Base(fileName)))
	if len(ext) > 16 {
		ext = "" // защита от мусорного «расширения»
	}
	stored := hex.EncodeToString(name) + ext
	if err := os.WriteFile(filepath.Join(dir, stored), data, 0o644); err != nil {
		return "", err
	}
	return subdir + "/" + stored, nil
}
