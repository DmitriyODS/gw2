package clients

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
)

// GeoIP — город по IP для карточки сеанса в профиле. Провайдер задаётся
// шаблоном URL (GEOIP_URL) с подстановками {ip} и {token}, ответ — JSON-объект
// с полем "city": так подходят и ip-api.com, и ipinfo.io, и ipapi.co — смена
// провайдера не требует правок кода.
//
// Результат кэшируется (одинаковый IP у всех входов пользователя), а любая
// ошибка означает лишь пустой город: карточка сеанса покажет IP.
type GeoIP struct {
	urlTemplate string
	token       string
	cache       CityCache
	http        *http.Client
	log         *slog.Logger
}

// CityCache — кэш «IP → город» (Redis). Реализуется repository/redisx.
type CityCache interface {
	Get(ctx context.Context, ip string) (string, bool)
	Set(ctx context.Context, ip, city string)
}

func NewGeoIP(urlTemplate, token string, cache CityCache, log *slog.Logger) *GeoIP {
	return &GeoIP{
		urlTemplate: urlTemplate,
		token:       token,
		cache:       cache,
		http:        &http.Client{Timeout: 4 * time.Second},
		log:         log,
	}
}

var _ domain.GeoResolver = (*GeoIP)(nil)

func (g *GeoIP) City(ctx context.Context, ip string) string {
	if g.urlTemplate == "" || ip == "" {
		return ""
	}
	if g.cache != nil {
		if city, ok := g.cache.Get(ctx, ip); ok {
			return city
		}
	}
	city := g.fetch(ctx, ip)
	// В кэш кладём и пустой ответ: не дёргать провайдера на каждый вход с
	// адреса, который он всё равно не знает.
	if g.cache != nil {
		g.cache.Set(ctx, ip, city)
	}
	return city
}

func (g *GeoIP) fetch(ctx context.Context, ip string) string {
	target := strings.NewReplacer(
		"{ip}", url.PathEscape(ip),
		"{token}", url.QueryEscape(g.token),
	).Replace(g.urlTemplate)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return ""
	}
	resp, err := g.http.Do(req)
	if err != nil {
		g.log.Debug("geoip.request_failed", "error", err)
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		g.log.Debug("geoip.bad_status", "status", resp.StatusCode)
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return ""
	}
	var out struct {
		City string `json:"city"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return ""
	}
	return strings.TrimSpace(out.City)
}
