package avatar

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
)

const avatarSubdir = "avatars"

// Storage — аватарки в UPLOAD_FOLDER/avatars (тот же volume, что у Flask;
// наружу файлы отдаёт nginx по /uploads/).
type Storage struct {
	root string
}

func NewStorage(uploadFolder string) *Storage { return &Storage{root: uploadFolder} }

// Save — проверка типа по содержимому (магические байты, как python-magic)
// и запись под случайным именем. Возвращает путь avatars/<hex>.<ext>.
func (s *Storage) Save(fileBytes []byte) (string, error) {
	var ext string
	switch http.DetectContentType(fileBytes) {
	case "image/jpeg":
		ext = "jpg"
	case "image/png":
		ext = "png"
	default:
		return "", domain.NewError("UPLOAD_ERROR", "Недопустимый тип файла. Разрешены: JPEG, PNG", 400)
	}

	name := make([]byte, 16)
	if _, err := rand.Read(name); err != nil {
		return "", err
	}
	dir := filepath.Join(s.root, avatarSubdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	filename := hex.EncodeToString(name) + "." + ext
	if err := os.WriteFile(filepath.Join(dir, filename), fileBytes, 0o644); err != nil {
		return "", err
	}
	return avatarSubdir + "/" + filename, nil
}

// Delete — молча игнорирует отсутствующий файл; путь вне uploads не трогаем.
func (s *Storage) Delete(avatarPath string) {
	if avatarPath == "" || strings.Contains(avatarPath, "..") {
		return
	}
	_ = os.Remove(filepath.Join(s.root, filepath.FromSlash(avatarPath)))
}

// ListFiles — все файлы каталога avatars/ для резервной копии (как
// avatars_dir.iterdir() во Flask: без рекурсии, отсутствующий каталог — пусто).
func (s *Storage) ListFiles() ([]domain.AvatarFile, error) {
	dir := filepath.Join(s.root, avatarSubdir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []domain.AvatarFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, domain.AvatarFile{Name: e.Name(), Data: data})
	}
	return out, nil
}

// WriteFile — восстановить файл аватарки из архива (имя — только basename,
// защита от zip-slip).
func (s *Storage) WriteFile(name string, data []byte) error {
	name = filepath.Base(filepath.FromSlash(name))
	if name == "" || name == "." || name == ".." {
		return nil
	}
	dir := filepath.Join(s.root, avatarSubdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), data, 0o644)
}

var _ domain.AvatarStorage = (*Storage)(nil)

// String — для логов старта.
func (s *Storage) String() string { return fmt.Sprintf("uploads:%s", s.root) }
