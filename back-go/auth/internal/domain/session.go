package domain

import (
	"context"
	"regexp"
	"strings"
	"time"
)

// Session — запись реестра активных входов (таблица user_sessions). Её id
// зашит в refresh-токен, поэтому «завершить сеанс» = проставить RevokedAt:
// следующий refresh этим токеном отвергается.
type Session struct {
	ID         int64
	UserID     int64
	CreatedAt  time.Time
	LastSeenAt time.Time
	Platform   string // mobile | desktop | web — иконка карточки
	Client     string // app | web — обёртка или браузер
	Device     string // модель или ОС: «Pixel 8», «MAC OS», «Chrome · Windows»
	UserAgent  string // сырой UA — на случай разбора незнакомого устройства
	IP         string
	City       string
}

// Опознание устройства по User-Agent.
const (
	PlatformMobile  = "mobile"
	PlatformDesktop = "desktop"
	PlatformWeb     = "web"

	ClientApp = "app"
	ClientWeb = "web"
)

// SessionStore — персистентность реестра сессий.
type SessionStore interface {
	// Create — новый вход; возвращает id для refresh-токена.
	Create(ctx context.Context, s *Session) (int64, error)
	// Get — сессия по id; nil — нет такой (удалена вместе с пользователем).
	Get(ctx context.Context, id int64) (*Session, error)
	// Touch — отметить активность (двигает last_seen_at). Возвращает false,
	// если сессии нет или она отозвана — тогда refresh отвергается.
	Touch(ctx context.Context, id, userID int64) (bool, error)
	// ListActive — живые сессии пользователя, свежие первыми.
	ListActive(ctx context.Context, userID int64) ([]*Session, error)
	// Revoke — завершить сеанс (только свой: userID в условии).
	Revoke(ctx context.Context, id, userID int64) error
	// SetCity — дозаполнить город после асинхронного гео-резолва.
	SetCity(ctx context.Context, id int64, city string) error
}

// GeoResolver — город по IP (внешний гео-сервис). Пустая строка — не
// определился: карточка сеанса просто покажет IP.
type GeoResolver interface {
	City(ctx context.Context, ip string) string
}

// SessionMeta — данные о запросе, из которых рождается сессия. Транспорт
// кладёт их в контекст (WithSessionMeta), сервис достаёт при выпуске
// refresh-токена. Сквозной параметр во всех use-case'ах, выдающих сессию, был
// бы шумом: это чисто транспортная примесь, как request-id.
type SessionMeta struct {
	UserAgent string
	IP        string
	// Refresh — refresh-токен из cookie запроса. Если он живой и принадлежит
	// тому же пользователю, новая пара токенов остаётся в ТОЙ ЖЕ сессии
	// (смена компании, change-default), а не плодит карточку устройства.
	Refresh string
}

type sessionMetaKey struct{}

func WithSessionMeta(ctx context.Context, m SessionMeta) context.Context {
	return context.WithValue(ctx, sessionMetaKey{}, m)
}

// SessionMetaFrom — примесь запроса; нулевая структура, если транспорт её не
// положил (тесты, gRPC-пути).
func SessionMetaFrom(ctx context.Context) SessionMeta {
	m, _ := ctx.Value(sessionMetaKey{}).(SessionMeta)
	return m
}

// «Linux; Android 14; Pixel 8 Build/…» — модель между версией ОС и Build.
var reAndroidModel = regexp.MustCompile(`Android[^;)]*;\s*([^;)]+?)(?:\s+Build/[^;)]*)?[;)]`)

// Порядок важен: Chrome и Safari присутствуют в UA почти всех браузеров, так
// что специфичные бренды проверяются раньше.
var browserMarks = []struct{ mark, name string }{
	{"YaBrowser/", "Яндекс.Браузер"},
	{"Edg/", "Edge"},
	{"OPR/", "Opera"},
	{"Firefox/", "Firefox"},
	{"Chrome/", "Chrome"},
	{"Safari/", "Safari"},
}

// DescribeUserAgent — опознать устройство по User-Agent: платформа (иконка),
// клиент (обёртка или браузер) и человекочитаемое устройство.
//
// Метки обёрток: GrooveWorkApp — Capacitor-обёртка (appendUserAgent), Electron —
// десктоп-клиент. Всё остальное — обычный браузер.
func DescribeUserAgent(ua string) (platform, client, device string) {
	switch {
	case strings.Contains(ua, "GrooveWorkApp"):
		return PlatformMobile, ClientApp, androidDevice(ua)
	case strings.Contains(ua, "Electron"):
		if os := osName(ua); os != "" {
			return PlatformDesktop, ClientApp, os
		}
		return PlatformDesktop, ClientApp, "Компьютер"
	default:
		return PlatformWeb, ClientWeb, webDevice(ua)
	}
}

func androidDevice(ua string) string {
	if m := reAndroidModel.FindStringSubmatch(ua); len(m) == 2 {
		if model := strings.TrimSpace(m[1]); model != "" && !strings.EqualFold(model, "wv") {
			return model
		}
	}
	if strings.Contains(ua, "iPhone") {
		return "iPhone"
	}
	if strings.Contains(ua, "iPad") {
		return "iPad"
	}
	return "Android"
}

func osName(ua string) string {
	switch {
	case strings.Contains(ua, "Mac OS"), strings.Contains(ua, "Macintosh"):
		return "MAC OS"
	case strings.Contains(ua, "Windows"):
		return "Windows"
	case strings.Contains(ua, "Linux"), strings.Contains(ua, "X11"):
		return "Linux"
	}
	return ""
}

// webDevice — «браузер · ОС» для подсказки на карточке веб-сеанса (сам
// заголовок карточки по макету — «Веб-приложение»).
func webDevice(ua string) string {
	browser := ""
	for _, b := range browserMarks {
		if strings.Contains(ua, b.mark) {
			browser = b.name
			break
		}
	}
	os := osName(ua)
	if strings.Contains(ua, "Android") || strings.Contains(ua, "iPhone") || strings.Contains(ua, "iPad") {
		os = androidDevice(ua)
	}
	switch {
	case browser != "" && os != "":
		return browser + " · " + os
	case browser != "":
		return browser
	default:
		return os
	}
}
