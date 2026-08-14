// Package files — файлы вложений мессенджера в хранилище (pkg/storage:
// локальный том или S3). Наружу файлы отдаёт nginx по /uploads/.
package files

import (
	"context"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/storage"
)

const messagesSubdir = "messages"

type Store struct {
	st storage.Storage
}

var _ domain.FileStore = (*Store)(nil)

func NewStore(st storage.Storage) *Store {
	return &Store{st: st}
}

// newKey — messages/YYYY/MM/{uuid32hex}{ext}; разделители всегда «/».
func (s *Store) newKey(ext string) string {
	now := time.Now().UTC()
	return fmt.Sprintf("%s/%04d/%02d/%s%s", messagesSubdir, now.Year(), int(now.Month()),
		strings.ReplaceAll(uuid.NewString(), "-", ""), ext)
}

func (s *Store) Save(data []byte, ext string) (string, error) {
	key := s.newKey(ext)
	if err := s.st.Put(context.Background(), key, data, contentType(ext)); err != nil {
		return "", err
	}
	return key, nil
}

// Copy — серверная копия (пересылка): удаление одной копии не задевает другую.
// SaveStream — запись потоком: вложение в сотни мегабайт нельзя подержать в
// памяти целиком. Размер известен заранее (его подтвердили принятые части).
func (s *Store) SaveStream(r io.Reader, ext string, size int64) (string, error) {
	key := s.newKey(ext)
	if err := s.st.PutStream(context.Background(), key, r, size, contentType(ext)); err != nil {
		return "", err
	}
	return key, nil
}

/* Приём файла частями (pkg/chunkupload.Store): части лежат обычными объектами
   во временном префиксе и собираются в поток на завершении. */

func (s *Store) PutRaw(ctx context.Context, key string, data []byte) error {
	return s.st.Put(ctx, key, data, "application/octet-stream")
}

func (s *Store) OpenStream(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.st.Open(ctx, key)
}

func (s *Store) RemoveKeys(ctx context.Context, keys ...string) {
	s.st.Remove(ctx, keys...)
}

func (s *Store) Copy(srcRelPath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(srcRelPath))
	if len(ext) > 16 {
		ext = ext[:16]
	}
	dstKey := s.newKey(ext)
	if err := s.st.Copy(context.Background(), srcRelPath, dstKey); err != nil {
		return "", err
	}
	return dstKey, nil
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
