// Package files — файлы вложений мессенджера в общем uploads-каталоге
// (UPLOAD_FOLDER, тот же volume, что у Flask; наружу отдаёт nginx /uploads/).
package files

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
)

const messagesSubdir = "messages"

type Store struct {
	root string
	log  *slog.Logger
}

var _ domain.FileStore = (*Store)(nil)

func NewStore(uploadFolder string, log *slog.Logger) *Store {
	return &Store{root: uploadFolder, log: log}
}

// relPath — messages/YYYY/MM/{uuid32hex}{ext}; разделители всегда «/»
// (как во Flask: rel_path.replace(os.sep, "/")).
func (s *Store) newRelPath(ext string) string {
	now := time.Now().UTC()
	return fmt.Sprintf("%s/%04d/%02d/%s%s", messagesSubdir, now.Year(), int(now.Month()),
		strings.ReplaceAll(uuid.NewString(), "-", ""), ext)
}

func (s *Store) abs(relPath string) string {
	return filepath.Join(s.root, filepath.FromSlash(relPath))
}

func (s *Store) Save(data []byte, ext string) (string, error) {
	relPath := s.newRelPath(ext)
	absPath := s.abs(relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(absPath, data, 0o644); err != nil {
		return "", err
	}
	return relPath, nil
}

// Copy — потоковая физическая копия (вложения бывают до 25 МБ).
func (s *Store) Copy(srcRelPath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(srcRelPath))
	if len(ext) > 16 {
		ext = ext[:16]
	}
	dstRel := s.newRelPath(ext)
	dstAbs := s.abs(dstRel)
	if err := os.MkdirAll(filepath.Dir(dstAbs), 0o755); err != nil {
		return "", err
	}

	src, err := os.Open(s.abs(srcRelPath))
	if err != nil {
		return "", err
	}
	defer src.Close()
	dst, err := os.Create(dstAbs)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(dstAbs)
		return "", err
	}
	if err := dst.Close(); err != nil {
		return "", err
	}
	return dstRel, nil
}

func (s *Store) Remove(paths []string) {
	for _, p := range paths {
		if p == "" {
			continue
		}
		if err := os.Remove(s.abs(p)); err != nil && !os.IsNotExist(err) {
			s.log.Warn("attachment.unlink_failed", "path", p, "error", err)
		}
	}
}
