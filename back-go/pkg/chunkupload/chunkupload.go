// Package chunkupload — общий для платформы приём больших файлов частями.
//
// Файл крупнее Threshold обязан ехать чанками в ЛЮБОМ разделе: одним запросом
// он упирается в таймауты прокси, не даёт клиенту показать прогресс и после
// обрыва сети начинается заново. Клиент заводит сессию, шлёт части по порядку
// и просит собрать файл.
//
// Устройство рассчитано на НЕСКОЛЬКО инстансов сервиса: сессия живёт в БД
// (upload_sessions), а части — обычными объектами во временном префиксе
// хранилища. Поэтому соседние части могут попасть на разные процессы, а сборка
// идёт потоком (storage.PutStream) — гигабайт не оказывается в памяти ни на
// одном шаге. Брошенные сессии (закрытая вкладка) убирает Sweep.
package chunkupload

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
	"github.com/DmitriyODS/gw2/back-go/pkg/records"
)

const (
	// Threshold — с какого размера файл обязан ехать частями. Ниже порога
	// разница незаметна, а лишний круг запросов только замедляет мелочь.
	Threshold = 10 << 20
	// ChunkSize — рекомендуемый размер части: его сервер сообщает клиенту при
	// заведении сессии. Мельче — прогресс дробнее, но круговых запросов больше.
	ChunkSize = 5 << 20
	// TTL — сколько сессия живёт без новых частей.
	TTL = 30 * time.Minute
	// sweepEvery — как часто убираем брошенные сессии.
	sweepEvery = 5 * time.Minute
)

// Store — что движку нужно от файлового хранилища сервиса. Реализуется
// pkg/records.FileStore; свои сторы сервисов добирают три метода.
type Store interface {
	// PutRaw — записать служебный объект под заданным ключом (часть загрузки).
	PutRaw(ctx context.Context, key string, data []byte) error
	// OpenStream — объект на чтение потоком.
	OpenStream(ctx context.Context, key string) (io.ReadCloser, error)
	// RemoveKeys — убрать объекты (части после сборки или отмены).
	RemoveKeys(ctx context.Context, keys ...string)
}

// Session — заведённая загрузка.
type Session struct {
	Code    string `json:"code"`
	Service string `json:"-"`
	// UserID — кто грузит (ему и только ему принадлежит сессия).
	UserID int64 `json:"-"`
	// CompanyID/QuotaUserID — чья квота платит за файл: ровно одно из двух.
	// Это не всегда загружающий — файл в чужом расшаренном разделе занимает
	// место его владельца.
	CompanyID   int64 `json:"-"`
	QuotaUserID int64 `json:"-"`
	// Scope — контекст раздела (id реестра, папки, переписки): смысл знает
	// только сам сервис.
	Scope     string `json:"scope"`
	FileName  string `json:"file_name"`
	Mime      string `json:"mime"`
	TotalSize int64  `json:"total_size"`
	ChunkSize int    `json:"chunk_size"`
	// Received — сколько частей принято подряд; докачка идёт со следующей.
	Received int `json:"received"`
}

// Chunks — сколько всего частей у файла этого размера.
func (s Session) Chunks() int {
	if s.ChunkSize <= 0 {
		return 0
	}
	return int((s.TotalSize + int64(s.ChunkSize) - 1) / int64(s.ChunkSize))
}

// Complete — все ли части на месте.
func (s Session) Complete() bool { return s.Received >= s.Chunks() }

// Manager — приём частей одного сервиса.
type Manager struct {
	pool    *pgxpool.Pool
	store   Store
	service string
	log     *slog.Logger
}

func New(pool *pgxpool.Pool, store Store, service string, log *slog.Logger) *Manager {
	return &Manager{pool: pool, store: store, service: service, log: log}
}

var (
	ErrNotFound = apierror.New("UPLOAD_NOT_FOUND", "Загрузка не найдена или устарела", 404)
	ErrOrder    = apierror.New("UPLOAD_CHUNK_ORDER", "Часть файла пришла не по порядку", 409)
	ErrSize     = apierror.New("UPLOAD_SIZE", "Размер файла не совпал с заявленным", 400)
)

// Init — завести сессию. Права на загрузку и потолок размера проверяет сервис
// ДО вызова: движок про его правила ничего не знает.
func (m *Manager) Init(ctx context.Context, s Session) (Session, error) {
	code, err := records.NewShareCode()
	if err != nil {
		return Session{}, err
	}
	s.Code, s.Service, s.ChunkSize, s.Received = code, m.service, ChunkSize, 0
	if _, err := m.pool.Exec(ctx, `
        INSERT INTO upload_sessions
            (code, service, user_id, company_id, quota_user_id, scope,
             file_name, mime, total_size, chunk_size)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		s.Code, s.Service, s.UserID, s.CompanyID, s.QuotaUserID, s.Scope,
		s.FileName, s.Mime, s.TotalSize, s.ChunkSize); err != nil {
		return Session{}, err
	}
	return s, nil
}

// Get — сессия по коду. Чужую не отдаём: код случайный, но загрузка привязана
// к тому, кто её начал.
func (m *Manager) Get(ctx context.Context, code string, userID int64) (Session, error) {
	var s Session
	err := m.pool.QueryRow(ctx, `
        SELECT code, service, user_id, company_id, quota_user_id, scope,
               file_name, mime, total_size, chunk_size, received
          FROM upload_sessions WHERE code = $1 AND service = $2 AND user_id = $3`,
		code, m.service, userID).Scan(&s.Code, &s.Service, &s.UserID, &s.CompanyID,
		&s.QuotaUserID, &s.Scope, &s.FileName, &s.Mime, &s.TotalSize, &s.ChunkSize, &s.Received)
	if err == pgx.ErrNoRows {
		return Session{}, ErrNotFound
	}
	return s, err
}

// Chunk — принять часть под номером index (с нуля). Номер обязан совпасть с
// уже принятым количеством: так повторно отправленная часть (клиент не дождался
// ответа и повторил) не задваивается, а разъехавшаяся видна сразу.
//
// Часть с ТЕКУЩИМ номером −1 считается повтором и подтверждается молча: иначе
// после потерянного ответа клиент вечно бодался бы с сервером.
func (m *Manager) Chunk(ctx context.Context, code string, userID int64, index int, data []byte) (Session, error) {
	s, err := m.Get(ctx, code, userID)
	if err != nil {
		return Session{}, err
	}
	if index == s.Received-1 {
		return s, nil
	}
	if index != s.Received {
		return s, ErrOrder
	}
	if int64(s.Received)*int64(s.ChunkSize)+int64(len(data)) > s.TotalSize {
		return s, ErrSize
	}
	if err := m.store.PutRaw(ctx, m.chunkKey(s.Code, index), data); err != nil {
		return s, err
	}
	// Счётчик двигаем условием по прежнему значению: две части с одним номером
	// от двух вкладок не должны сойти за две разные.
	tag, err := m.pool.Exec(ctx, `
        UPDATE upload_sessions SET received = $1, updated_at = now()
         WHERE code = $2 AND received = $3`, index+1, s.Code, s.Received)
	if err != nil {
		return s, err
	}
	if tag.RowsAffected() == 0 {
		return m.Get(ctx, code, userID)
	}
	s.Received = index + 1
	return s, nil
}

// Reader — собранное содержимое загрузки потоком: части открываются по одной
// по мере чтения. Закрыть вызывающему.
func (m *Manager) Reader(ctx context.Context, s Session) io.ReadCloser {
	return &chunkReader{ctx: ctx, m: m, session: s}
}

// Done — убрать части и саму сессию (файл уже собран).
func (m *Manager) Done(ctx context.Context, s Session) {
	keys := make([]string, 0, s.Received)
	for i := 0; i < s.Received; i++ {
		keys = append(keys, m.chunkKey(s.Code, i))
	}
	m.store.RemoveKeys(ctx, keys...)
	if _, err := m.pool.Exec(ctx, `DELETE FROM upload_sessions WHERE code = $1`, s.Code); err != nil {
		m.log.Warn("chunkupload.session_cleanup_failed", "code", s.Code, "error", err)
	}
}

// Cancel — отменить загрузку по требованию клиента.
func (m *Manager) Cancel(ctx context.Context, code string, userID int64) error {
	s, err := m.Get(ctx, code, userID)
	if err != nil {
		return err
	}
	m.Done(ctx, s)
	return nil
}

// Sweep — фоновая уборка брошенных сессий: закрытая вкладка не должна
// оставлять части в хранилище навсегда.
func (m *Manager) Sweep(ctx context.Context) {
	ticker := time.NewTicker(sweepEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.sweepOnce(ctx)
		}
	}
}

func (m *Manager) sweepOnce(ctx context.Context) {
	rows, err := m.pool.Query(ctx, `
        SELECT code, received FROM upload_sessions
         WHERE service = $1 AND updated_at < now() - $2::interval`,
		m.service, TTL.String())
	if err != nil {
		m.log.Warn("chunkupload.sweep_failed", "error", err)
		return
	}
	type stale struct {
		code     string
		received int
	}
	var list []stale
	for rows.Next() {
		var s stale
		if err := rows.Scan(&s.code, &s.received); err != nil {
			break
		}
		list = append(list, s)
	}
	rows.Close()

	for _, s := range list {
		m.Done(ctx, Session{Code: s.code, Received: s.received})
	}
	if len(list) > 0 {
		m.log.Info("chunkupload.swept", "service", m.service, "count", len(list))
	}
}

// chunkKey — часть загрузки во временном префиксе сервиса.
func (m *Manager) chunkKey(code string, index int) string {
	return fmt.Sprintf("%s/tmp/%s/%06d", m.service, code, index)
}

// chunkReader — последовательное чтение частей как одного потока.
type chunkReader struct {
	ctx     context.Context
	m       *Manager
	session Session
	index   int
	current io.ReadCloser
}

func (r *chunkReader) Read(p []byte) (int, error) {
	for {
		if r.current == nil {
			if r.index >= r.session.Received {
				return 0, io.EOF
			}
			rc, err := r.m.store.OpenStream(r.ctx, r.m.chunkKey(r.session.Code, r.index))
			if err != nil {
				return 0, err
			}
			r.current = rc
		}
		n, err := r.current.Read(p)
		if n > 0 {
			return n, nil
		}
		if err == io.EOF {
			r.current.Close()
			r.current = nil
			r.index++
			continue
		}
		if err != nil {
			return 0, err
		}
	}
}

func (r *chunkReader) Close() error {
	if r.current != nil {
		err := r.current.Close()
		r.current = nil
		return err
	}
	return nil
}
