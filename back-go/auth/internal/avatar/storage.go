package avatar

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/storage"
)

const avatarSubdir = "avatars"

// Storage — аватарки в хранилище (pkg/storage: локальный том или S3) под
// префиксом avatars/. Наружу файлы отдаёт nginx по /uploads/.
type Storage struct {
	st storage.Storage
}

func NewStorage(st storage.Storage) *Storage { return &Storage{st: st} }

// Save — проверка типа по содержимому (магические байты, как python-magic)
// и запись под случайным именем. Возвращает путь avatars/<hex>.<ext>.
func (s *Storage) Save(fileBytes []byte) (string, error) {
	var ext, ct string
	switch http.DetectContentType(fileBytes) {
	case "image/jpeg":
		ext, ct = "jpg", "image/jpeg"
	case "image/png":
		ext, ct = "png", "image/png"
	default:
		return "", domain.NewError("UPLOAD_ERROR", "Недопустимый тип файла. Разрешены: JPEG, PNG", 400)
	}

	name := make([]byte, 16)
	if _, err := rand.Read(name); err != nil {
		return "", err
	}
	key := avatarSubdir + "/" + hex.EncodeToString(name) + "." + ext
	if err := s.st.Put(context.Background(), key, fileBytes, ct); err != nil {
		return "", err
	}
	return key, nil
}

// Delete — молча игнорирует отсутствующий файл; путь вне avatars/ не трогаем.
func (s *Storage) Delete(avatarPath string) {
	if avatarPath == "" || strings.Contains(avatarPath, "..") {
		return
	}
	s.st.Remove(context.Background(), avatarPath)
}

// ListFiles — все файлы avatars/ для резервной копии. Name — basename ключа.
func (s *Storage) ListFiles() ([]domain.AvatarFile, error) {
	ctx := context.Background()
	keys, err := s.st.List(ctx, avatarSubdir)
	if err != nil {
		return nil, err
	}
	var out []domain.AvatarFile
	for _, key := range keys {
		rc, err := s.st.Open(ctx, key)
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		out = append(out, domain.AvatarFile{Name: path.Base(key), Data: data})
	}
	return out, nil
}

// WriteFile — восстановить файл аватарки из архива (имя — только basename,
// защита от zip-slip).
func (s *Storage) WriteFile(name string, data []byte) error {
	name = path.Base(strings.ReplaceAll(name, "\\", "/"))
	if name == "" || name == "." || name == ".." {
		return nil
	}
	return s.st.Put(context.Background(), avatarSubdir+"/"+name, data, "")
}

var _ domain.AvatarStorage = (*Storage)(nil)
