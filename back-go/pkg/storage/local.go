package storage

import (
	"context"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// localStore — файлы на диске под общим корнем (dev / прежнее поведение
// uploads-тома). contentType игнорируется (mime раздаёт nginx по расширению).
type localStore struct {
	root string
	log  *slog.Logger
}

func NewLocal(root string, log *slog.Logger) Storage {
	return &localStore{root: root, log: log}
}

// abs — абсолютный путь по ключу; ключ всегда со «/» в качестве разделителя.
func (s *localStore) abs(key string) string {
	return filepath.Join(s.root, filepath.FromSlash(key))
}

func (s *localStore) Put(_ context.Context, key string, data []byte, _ string) error {
	abs := s.abs(key)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	return os.WriteFile(abs, data, 0o644)
}

func (s *localStore) Open(_ context.Context, key string) (io.ReadCloser, error) {
	return os.Open(s.abs(key))
}

func (s *localStore) Copy(_ context.Context, srcKey, dstKey string) error {
	src, err := os.Open(s.abs(srcKey))
	if err != nil {
		return err
	}
	defer src.Close()

	dstAbs := s.abs(dstKey)
	if err := os.MkdirAll(filepath.Dir(dstAbs), 0o755); err != nil {
		return err
	}
	dst, err := os.Create(dstAbs)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(dstAbs)
		return err
	}
	return dst.Close()
}

func (s *localStore) Remove(_ context.Context, keys ...string) {
	for _, k := range keys {
		if k == "" || strings.Contains(k, "..") {
			continue
		}
		if err := os.Remove(s.abs(k)); err != nil && !os.IsNotExist(err) {
			s.log.Warn("storage.remove_failed", "key", k, "error", err)
		}
	}
}

func (s *localStore) List(_ context.Context, prefix string) ([]string, error) {
	var keys []string
	err := filepath.WalkDir(s.abs(prefix), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil // отсутствующий каталог префикса — пусто
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(s.root, path)
		if err != nil {
			return err
		}
		keys = append(keys, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return keys, nil
}
