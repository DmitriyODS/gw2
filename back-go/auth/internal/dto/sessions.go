package dto

import "github.com/DmitriyODS/gw2/back-go/auth/internal/domain"

// SessionInfo — карточка входа в разделе «Авторизация и сессии» профиля.
type SessionInfo struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`    // «Pixel 8, приложение», «Веб-приложение»
	Platform string `json:"platform"` // mobile | desktop | web — иконка карточки
	Device   string `json:"device"`   // подробность для подсказки: «Chrome · Windows»
	City     string `json:"city"`     // пусто — гео не определилось
	IP       string `json:"ip"`

	CreatedAt  JSONTime `json:"created_at"`
	LastSeenAt JSONTime `json:"last_seen_at"`
	// Current — этот сеанс и есть текущий (его завершение = выход из системы).
	Current bool `json:"current"`
}

// sessionTitle — заголовок карточки: у обёрток это устройство, у браузера —
// нейтральное «Веб-приложение» (модель браузера уходит в подсказку).
func sessionTitle(s *domain.Session) string {
	if s.Client == domain.ClientApp && s.Device != "" {
		return s.Device + ", приложение"
	}
	return "Веб-приложение"
}

func NewSessionInfo(s *domain.Session, current bool) SessionInfo {
	return SessionInfo{
		ID:         s.ID,
		Title:      sessionTitle(s),
		Platform:   s.Platform,
		Device:     s.Device,
		City:       s.City,
		IP:         s.IP,
		CreatedAt:  JSONTime(s.CreatedAt),
		LastSeenAt: JSONTime(s.LastSeenAt),
		Current:    current,
	}
}
