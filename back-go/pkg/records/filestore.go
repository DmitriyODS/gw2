package records

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"mime"
	"path/filepath"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/pkg/billingclient"
	"github.com/DmitriyODS/gw2/back-go/pkg/storage"
)

// QuotaTracker — учёт занятого места (billingsvc). Реализация —
// *billingclient.Client; nil-клиент разрешает всё, поэтому проверки здесь
// безусловные.
type QuotaTracker interface {
	// EnsureStorage — влезает ли файл в квоту владельца (companyID > 0 —
	// квота создателя компании).
	EnsureStorage(ctx context.Context, userID, companyID, bytes int64) error
	// TrackStorage — сдвинуть занятое место и пополнить журнал файлов.
	TrackStorage(ctx context.Context, userID, companyID int64, ch billingclient.StorageChange)
}

// FileStore — запись загруженных файлов/картинок записей в хранилище
// (pkg/storage: локальный том или S3). Ключ на диске случайный (без утечки
// исходного имени), оригинальное имя хранится в метаданных записи. subdir —
// префикс сервиса ("registry"/"calendar").
//
// Квота хранилища считается ЗДЕСЬ, в одной точке для всех сервисов на этом
// сторе: SaveFor проверяет остаток и прибавляет размер, RemoveFor измеряет
// удаляемые объекты и вычитает. Без подключённого трекера обе операции
// работают как раньше — просто без учёта.
type FileStore struct {
	st      storage.Storage
	subdir  string
	quota   QuotaTracker
	service string
}

func NewFileStore(st storage.Storage, subdir string) *FileStore {
	return &FileStore{st: st, subdir: subdir, service: subdir}
}

// WithQuota — подключить учёт занятого места. service — раздел в разбивке
// хранилища (обычно совпадает с subdir).
func (s *FileStore) WithQuota(quota QuotaTracker, service string) *FileStore {
	s.quota = quota
	if service != "" {
		s.service = service
	}
	return s
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

// SaveFor — то же, что Save, но с учётом квоты владельца: файл сверх лимита
// не записывается вовсе, записанный увеличивает занятое место. companyID > 0 —
// файл компании: место тратится из квоты её создателя.
//
// Заодно файл попадает в журнал хранилища. Сущность-носитель здесь неизвестна
// (файл нередко грузится раньше неё) — ссылку проставит ближайшая сверка
// биллинга с владельцем, а имя и размер известны уже сейчас.
func (s *FileStore) SaveFor(ctx context.Context, userID, companyID int64, fileName string, data []byte) (string, error) {
	if s.quota != nil {
		if err := s.quota.EnsureStorage(ctx, userID, companyID, int64(len(data))); err != nil {
			return "", err
		}
	}
	key, err := s.Save(fileName, data)
	if err != nil {
		return "", err
	}
	if s.quota != nil {
		s.quota.TrackStorage(ctx, userID, companyID, billingclient.StorageChange{
			Service: s.service,
			Added: []billingclient.StoredFile{{
				Key: key, Name: fileName, Size: int64(len(data)),
			}},
		})
	}
	return key, nil
}

func (s *FileStore) Remove(paths []string) {
	s.st.Remove(context.Background(), paths...)
}

// RemoveFor — удаление с возвратом места в квоту. Размеры мерить не нужно: их
// знает журнал биллинга, поэтому здесь уходят только ключи (иначе на каждое
// удаление приходился бы запрос в S3 на объект).
func (s *FileStore) RemoveFor(ctx context.Context, userID, companyID int64, paths []string) {
	if len(paths) == 0 {
		return
	}
	s.st.Remove(ctx, paths...)
	if s.quota != nil {
		s.quota.TrackStorage(ctx, userID, companyID, billingclient.StorageChange{
			Service: s.service,
			Removed: paths,
		})
	}
}

// Open — прочитать содержимое объекта по ключу (для встраивания картинок в
// экспорт). Возвращает байты целиком.
func (s *FileStore) Open(key string) ([]byte, error) {
	rc, err := s.st.Open(context.Background(), key)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func contentType(ext string) string {
	if ct := mime.TypeByExtension(ext); ct != "" {
		return ct
	}
	return "application/octet-stream"
}
