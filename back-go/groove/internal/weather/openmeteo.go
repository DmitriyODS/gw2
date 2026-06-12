// Package weather — клиент Open-Meteo: текущая погода и геокодинг городов.
// Бесплатное API без ключа (https://open-meteo.com), коды погоды — WMO.
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/domain"
)

const (
	defaultForecastURL = "https://api.open-meteo.com/v1/forecast"
	defaultGeocodeURL  = "https://geocoding-api.open-meteo.com/v1/search"
	requestTimeout     = 8 * time.Second
)

type Client struct {
	http        *http.Client
	log         *slog.Logger
	ForecastURL string // переопределяются в тестах (httptest)
	GeocodeURL  string
}

var _ domain.WeatherProvider = (*Client)(nil)

func New(log *slog.Logger) *Client {
	return &Client{
		http: &http.Client{
			Timeout: requestTimeout,
			// Только IPv4: у geocoding-api.open-meteo.com есть AAAA-запись,
			// и на хостах с полуживым IPv6 (ULA-адрес без глобального
			// маршрута) connect по ней «удаётся», а write падает — happy
			// eyeballs от этого не спасает. По IPv4 Open-Meteo доступен
			// полностью.
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
					return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp4", addr)
				},
			},
		},
		log:         log,
		ForecastURL: defaultForecastURL,
		GeocodeURL:  defaultGeocodeURL,
	}
}

func (c *Client) getJSON(ctx context.Context, rawURL string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("open-meteo: status %d", resp.StatusCode)
	}
	return json.Unmarshal(body, out)
}

func (c *Client) Current(ctx context.Context, lat, lon float64) (*domain.Weather, error) {
	q := url.Values{}
	q.Set("latitude", strconv.FormatFloat(lat, 'f', 4, 64))
	q.Set("longitude", strconv.FormatFloat(lon, 'f', 4, 64))
	q.Set("current", "temperature_2m,weather_code,wind_speed_10m,is_day")

	var payload struct {
		Current struct {
			Temperature float64 `json:"temperature_2m"`
			WeatherCode int     `json:"weather_code"`
			WindSpeed   float64 `json:"wind_speed_10m"`
			IsDay       int     `json:"is_day"`
		} `json:"current"`
	}
	if err := c.getJSON(ctx, c.ForecastURL+"?"+q.Encode(), &payload); err != nil {
		return nil, err
	}
	return &domain.Weather{
		Code:    payload.Current.WeatherCode,
		TempC:   payload.Current.Temperature,
		WindKmh: payload.Current.WindSpeed,
		IsDay:   payload.Current.IsDay == 1,
	}, nil
}

func (c *Client) SearchCities(ctx context.Context, query string, count int) ([]domain.GeoPlace, error) {
	q := url.Values{}
	q.Set("name", query)
	q.Set("count", strconv.Itoa(count))
	q.Set("language", "ru")
	q.Set("format", "json")

	var payload struct {
		Results []struct {
			Name      string  `json:"name"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Country   *string `json:"country"`
			Admin1    *string `json:"admin1"`
		} `json:"results"`
	}
	if err := c.getJSON(ctx, c.GeocodeURL+"?"+q.Encode(), &payload); err != nil {
		return nil, err
	}
	out := make([]domain.GeoPlace, 0, len(payload.Results))
	for _, r := range payload.Results {
		out = append(out, domain.GeoPlace{
			Name:    r.Name,
			Region:  r.Admin1,
			Country: r.Country,
			Lat:     r.Latitude,
			Lon:     r.Longitude,
		})
	}
	return out, nil
}
