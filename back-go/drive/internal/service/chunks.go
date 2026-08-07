package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"sync"
	"time"

	"github.com/DmitriyODS/gw2/back-go/drive/internal/domain"
)

/* Загрузка большими частями.

   Файл на сотни мегабайт одним запросом — это отсутствие прогресса докачки,
   упор в таймауты прокси и полный объём в памяти сервиса. Поэтому клиент
   заводит загрузку, шлёт куски по порядку и в конце просит собрать файл.

   Куски копятся во ВРЕМЕННОМ ФАЙЛЕ, а не в памяти: 500 МБ на процесс сервиса
   недопустимы даже в одном экземпляре. Отсюда инвариант: все куски одной
   загрузки обязаны попасть на ОДИН инстанс drivesvc — при масштабировании
   раздела понадобится липкая сессия по upload_id либо общее временное
   хранилище. Брошенные загрузки чистит фоновый цикл по времени последнего
   куска: закрытая вкладка не должна оставлять мусор на диске.  */

const (
	// UploadTTL — сколько живёт загрузка без новых кусков.
	UploadTTL = 30 * time.Minute
	// uploadSweep — как часто убираем брошенные.
	uploadSweep = 5 * time.Minute
)

// Upload — идущая загрузка по частям.
type Upload struct {
	ID       string
	UserID   int64
	Name     string
	Mime     string
	Size     int64 // ожидаемый полный размер
	FolderID *int64

	mu       sync.Mutex
	file     *os.File
	received int64
	touched  time.Time
}

// Uploads — реестр идущих загрузок.
type Uploads struct {
	mu   sync.Mutex
	byID map[string]*Upload
	dir  string
}

func NewUploads(dir string) *Uploads {
	return &Uploads{byID: map[string]*Upload{}, dir: dir}
}

var errUnknownUpload = domain.ErrNotFound

// BeginUpload — завести загрузку: проверяем права на папку и размер, дальше
// принимаем куски. Место в хранилище проверяется на сборке — до неё файла
// ещё нет, а держать бронь квоты по всем брошенным загрузкам дороже.
func (s *Service) BeginUpload(ctx context.Context, userID int64, name, mimeType string, size int64, folderID *int64) (string, error) {
	if size <= 0 {
		return "", domain.ErrEmptyFile
	}
	if size > domain.MaxFileSize {
		return "", domain.ErrFileTooBig
	}
	if _, err := s.uploadOwner(ctx, userID, folderID); err != nil {
		return "", err
	}

	f, err := os.CreateTemp(s.uploads.dir, "gw2-drive-*")
	if err != nil {
		return "", err
	}
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		f.Close()           //nolint:errcheck
		os.Remove(f.Name()) //nolint:errcheck
		return "", err
	}
	up := &Upload{
		ID: hex.EncodeToString(buf), UserID: userID, Name: name, Mime: mimeType,
		Size: size, FolderID: folderID, file: f, touched: time.Now(),
	}
	s.uploads.mu.Lock()
	s.uploads.byID[up.ID] = up
	s.uploads.mu.Unlock()
	return up.ID, nil
}

// WriteChunk — дописать кусок. offset обязателен и обязан совпасть с уже
// принятым объёмом: так повторно отправленный кусок (сеть моргнула, клиент
// повторил) не задваивается, а разъехавшийся — виден сразу.
func (s *Service) WriteChunk(userID int64, uploadID string, offset int64, r io.Reader) (int64, error) {
	up, err := s.upload(userID, uploadID)
	if err != nil {
		return 0, err
	}
	up.mu.Lock()
	defer up.mu.Unlock()

	if offset != up.received {
		// Клиент отстал или забежал вперёд — пусть продолжит с нашей отметки.
		return up.received, domain.ErrChunkOffset
	}
	written, err := io.Copy(up.file, io.LimitReader(r, domain.MaxFileSize))
	if err != nil {
		return up.received, err
	}
	up.received += written
	up.touched = time.Now()
	if up.received > up.Size {
		return up.received, domain.ErrFileTooBig
	}
	return up.received, nil
}

// FinishUpload — собрать файл из принятых кусков.
func (s *Service) FinishUpload(ctx context.Context, userID int64, uploadID string) (*domain.File, error) {
	up, err := s.upload(userID, uploadID)
	if err != nil {
		return nil, err
	}
	defer s.dropUpload(uploadID)

	up.mu.Lock()
	defer up.mu.Unlock()
	if up.received == 0 {
		return nil, domain.ErrEmptyFile
	}
	if _, err := up.file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	// Полный объём читаем в память ровно один раз — на записи в хранилище:
	// у storage.Put тело идёт байтами, потокового пути в контракте нет.
	data, err := io.ReadAll(up.file)
	if err != nil {
		return nil, err
	}
	return s.Upload(ctx, userID, up.Name, data, up.Mime, up.FolderID)
}

// CancelUpload — отменить загрузку и убрать временный файл.
func (s *Service) CancelUpload(userID int64, uploadID string) error {
	if _, err := s.upload(userID, uploadID); err != nil {
		return err
	}
	s.dropUpload(uploadID)
	return nil
}

func (s *Service) upload(userID int64, uploadID string) (*Upload, error) {
	s.uploads.mu.Lock()
	defer s.uploads.mu.Unlock()
	up, ok := s.uploads.byID[uploadID]
	if !ok || up.UserID != userID {
		return nil, errUnknownUpload
	}
	return up, nil
}

func (s *Service) dropUpload(uploadID string) {
	s.uploads.mu.Lock()
	up, ok := s.uploads.byID[uploadID]
	delete(s.uploads.byID, uploadID)
	s.uploads.mu.Unlock()
	if ok {
		up.close()
	}
}

func (u *Upload) close() {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.file == nil {
		return
	}
	name := u.file.Name()
	u.file.Close()  //nolint:errcheck
	os.Remove(name) //nolint:errcheck
	u.file = nil
}

// SweepUploads — фоновая чистка брошенных загрузок.
func (s *Service) SweepUploads(ctx context.Context) {
	ticker := time.NewTicker(uploadSweep)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sweepUploadsOnce()
		}
	}
}

func (s *Service) sweepUploadsOnce() {
	deadline := time.Now().Add(-UploadTTL)
	s.uploads.mu.Lock()
	stale := []*Upload{}
	for id, up := range s.uploads.byID {
		up.mu.Lock()
		old := up.touched.Before(deadline)
		up.mu.Unlock()
		if old {
			stale = append(stale, up)
			delete(s.uploads.byID, id)
		}
	}
	s.uploads.mu.Unlock()
	for _, up := range stale {
		up.close()
	}
	if len(stale) > 0 {
		s.log.Info("drive.uploads_swept", "count", len(stale))
	}
}

// uploadOwner — владелец будущего файла и проверка права писать в папку: та же
// логика, что и у обычной загрузки, но до приёма единого байта.
func (s *Service) uploadOwner(ctx context.Context, userID int64, folderID *int64) (int64, error) {
	if folderID == nil {
		return userID, nil
	}
	folder, err := s.repo.GetFolder(ctx, *folderID)
	if err != nil {
		return 0, err
	}
	if folder == nil || folder.DeletedAt != nil {
		return 0, domain.ErrNotFound
	}
	access, err := s.folderAccess(ctx, folder, userID)
	if err != nil {
		return 0, err
	}
	if !domain.AccessAtLeast(access, domain.AccessEdit) {
		return 0, domain.ErrForbidden
	}
	return folder.OwnerID, nil
}
