package service

import (
	"context"
	"net"
	"time"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
)

// Реестр входов: каждая выданная пара токенов принадлежит строке user_sessions,
// её id зашит в refresh. Отсюда — список устройств в профиле и «завершить
// сеанс». Весь реестр работает fail-open: любая его ошибка не должна мешать
// человеку войти, поэтому неудача возвращает sessionID = 0 (вход без карточки).

// geoTimeout — фон на определение города; вход его не ждёт.
const geoTimeout = 5 * time.Second

var errSessionNotFound = domain.NewError("NOT_FOUND", "Сеанс не найден", 404)

// ensureSessionID — id сессии для новой пары токенов: продолжаем текущую (если
// запрос пришёл с живым refresh того же пользователя — смена компании, смена
// пароля) либо заводим новую карточку устройства.
func (s *Service) ensureSessionID(ctx context.Context, userID int64) int64 {
	if s.sessions == nil {
		return 0
	}
	meta := domain.SessionMetaFrom(ctx)

	if meta.Refresh != "" {
		// Чужой или протухший refresh игнорируем: вход с этого устройства под
		// другим аккаунтом обязан завести свою карточку.
		if uid, _, sid, err := s.tokens.ParseRefresh(meta.Refresh); err == nil && uid == userID && sid > 0 {
			if alive, err := s.sessions.Touch(ctx, sid, userID); err == nil && alive {
				return sid
			}
		}
	}

	platform, client, device := domain.DescribeUserAgent(meta.UserAgent)
	sess := &domain.Session{
		UserID:    userID,
		Platform:  platform,
		Client:    client,
		Device:    device,
		UserAgent: meta.UserAgent,
		IP:        meta.IP,
	}
	id, err := s.sessions.Create(ctx, sess)
	if err != nil {
		s.log.Warn("session.create_failed", "user_id", userID, "error", err)
		return 0
	}
	s.resolveCityAsync(id, meta.IP)
	return id
}

// resolveCityAsync — город по IP отдельно от входа: внешний гео-сервис не
// должен добавлять свою задержку к логину, а не определившийся город просто
// остаётся пустым.
func (s *Service) resolveCityAsync(sessionID int64, ip string) {
	if s.geo == nil || !publicIP(ip) {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), geoTimeout)
		defer cancel()
		city := s.geo.City(ctx, ip)
		if city == "" {
			return
		}
		if err := s.sessions.SetCity(ctx, sessionID, city); err != nil {
			s.log.Warn("session.set_city_failed", "session_id", sessionID, "error", err)
		}
	}()
}

// publicIP — есть ли смысл спрашивать гео: серые и петлевые адреса внешний
// сервис всё равно не разрешит.
func publicIP(raw string) bool {
	ip := net.ParseIP(raw)
	return ip != nil && !ip.IsLoopback() && !ip.IsPrivate() &&
		!ip.IsLinkLocalUnicast() && !ip.IsUnspecified()
}

// checkSession — refresh по сессионному токену: живая сессия двигает
// last_seen_at, отозванная получает отказ (тот же код, что у битого токена —
// фронт уже умеет по нему разлогинивать).
func (s *Service) checkSession(ctx context.Context, sessionID, userID int64) error {
	if s.sessions == nil || sessionID == 0 {
		return nil
	}
	alive, err := s.sessions.Touch(ctx, sessionID, userID)
	if err != nil {
		s.log.Warn("session.touch_failed", "session_id", sessionID, "error", err)
		return nil // сбой БД не должен разлогинивать всех
	}
	if !alive {
		return domain.NewError("INVALID_TOKEN", "Сеанс завершён на другом устройстве", 401)
	}
	return nil
}

// ListSessions — активные входы пользователя для профиля; текущий помечен по
// refresh-cookie запроса.
func (s *Service) ListSessions(ctx context.Context, userID int64) ([]dto.SessionInfo, error) {
	if s.sessions == nil {
		return []dto.SessionInfo{}, nil
	}
	items, err := s.sessions.ListActive(ctx, userID)
	if err != nil {
		return nil, err
	}
	current := s.currentSessionID(ctx, userID)

	out := make([]dto.SessionInfo, 0, len(items))
	for _, it := range items {
		out = append(out, dto.NewSessionInfo(it, it.ID == current))
	}
	return out, nil
}

// RevokeSession — завершить сеанс. Только свой: чужой id не найдётся. Ответ об
// успехе тут обязан быть правдивым — «сеанс завершён» на чужой карточке значило
// бы, что человек считает устройство отключённым, а оно продолжает работать.
func (s *Service) RevokeSession(ctx context.Context, userID, sessionID int64) error {
	if s.sessions == nil {
		return nil
	}
	sess, err := s.sessions.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	if sess == nil || sess.UserID != userID {
		return errSessionNotFound
	}
	return s.sessions.Revoke(ctx, sessionID, userID)
}

// RevokeCurrentSession — выход из системы: карточка этого устройства уходит из
// списка, а его refresh перестаёт работать сразу (не дожидаясь 30 дней).
func (s *Service) RevokeCurrentSession(ctx context.Context, userID int64) error {
	if s.sessions == nil {
		return nil
	}
	if sid := s.currentSessionID(ctx, userID); sid > 0 {
		return s.sessions.Revoke(ctx, sid, userID)
	}
	return nil
}

/* RevokeOtherSessions — выйти на всех устройствах, кроме текущего. Зовётся и
   по кнопке в аккаунте, и автоматически при смене пароля: сменивший пароль
   рассчитывает, что прежние входы перестали работать — иначе украденный
   пароль остаётся рабочим до истечения чужого refresh (тридцать дней). */
func (s *Service) RevokeOtherSessions(ctx context.Context, userID int64) (int, error) {
	if s.sessions == nil {
		return 0, nil
	}
	return s.sessions.RevokeOthers(ctx, userID, s.currentSessionID(ctx, userID))
}

// currentSessionID — сессия, из которой пришёл запрос (по refresh-cookie);
// 0 — cookie нет, она чужая или легаси-токен без реестра.
func (s *Service) currentSessionID(ctx context.Context, userID int64) int64 {
	meta := domain.SessionMetaFrom(ctx)
	if meta.Refresh == "" {
		return 0
	}
	uid, _, sid, err := s.tokens.ParseRefresh(meta.Refresh)
	if err != nil || uid != userID {
		return 0
	}
	return sid
}
