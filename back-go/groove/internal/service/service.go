// Package service — бизнес-логика «Моего Groove» на портах домена.
// Порт прежних feed_service / pet_service / groove_ai_service Flask:
// поведение, лимиты, тексты и ключи Redis сохранены 1-в-1.
package service

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/domain"
)

type Service struct {
	feed      domain.FeedRepo
	pets      domain.PetRepo
	users     domain.UserReader
	companies domain.CompanyReader
	work      domain.WorkReader
	convs     domain.ConversationReader
	locs      domain.LocationRepo
	daily     domain.Daily
	pub       domain.EventPublisher
	ai        domain.AIClient
	msgr      domain.MessengerClient
	weather   domain.WeatherProvider
	log       *slog.Logger
}

func New(feed domain.FeedRepo, pets domain.PetRepo, users domain.UserReader,
	companies domain.CompanyReader, work domain.WorkReader,
	convs domain.ConversationReader, locs domain.LocationRepo, daily domain.Daily,
	pub domain.EventPublisher, ai domain.AIClient, msgr domain.MessengerClient,
	weather domain.WeatherProvider, log *slog.Logger) *Service {

	return &Service{
		feed: feed, pets: pets, users: users, companies: companies, work: work,
		convs: convs, locs: locs, daily: daily, pub: pub, ai: ai, msgr: msgr,
		weather: weather, log: log,
	}
}

// ─────────────────────────── время (МСК) ───────────────────────────

// todayMSK — текущая дата по Москве (полночь UTC-времени для date-колонок).
func todayMSK() time.Time {
	now := time.Now().In(domain.MSK)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// weekStartMSK — понедельник текущей недели (МСК).
func weekStartMSK() time.Time {
	today := todayMSK()
	offset := (int(today.Weekday()) + 6) % 7 // Пн=0 … Вс=6
	return today.AddDate(0, 0, -offset)
}

// mskMidnight — полночь даты d в таймзоне МСК (границы суток рейда).
func mskMidnight(d time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, domain.MSK)
}

// pyWeekday — день недели как в Python: Пн=0 … Вс=6.
func pyWeekday(d time.Time) int {
	return (int(d.Weekday()) + 6) % 7
}

// pythonOrdinal — date.toordinal() Python (дни от 0001-01-01, с единицы):
// детерминированный выбор квеста должен совпасть с прежним. Через номер
// юлианского дня — time.Duration на таких интервалах переполняется.
func pythonOrdinal(d time.Time) int {
	y, m, day := d.Date()
	a := (14 - int(m)) / 12
	yy := y + 4800 - a
	mm := int(m) + 12*a - 3
	jdn := day + (153*mm+2)/5 + 365*yy + yy/4 - yy/100 + yy/400 - 32045
	return jdn - 1721425 // JDN(0001-01-01) = 1721426 → ordinal 1
}

func isWeekend(d time.Time, weekend []int) bool {
	wd := pyWeekday(d)
	for _, w := range weekend {
		if w == wd {
			return true
		}
	}
	return false
}

// workingDaysBetween — число рабочих дней в интервале (start, end].
func workingDaysBetween(start, end time.Time, weekend []int) int {
	if !end.After(start) || len(weekend) >= 7 {
		return 0
	}
	n := 0
	for d := start.AddDate(0, 0, 1); !d.After(end); d = d.AddDate(0, 0, 1) {
		if !isWeekend(d, weekend) {
			n++
		}
	}
	return n
}

func (s *Service) weekendDays(ctx context.Context, companyID int64) []int {
	days, err := s.companies.WeekendDays(ctx, companyID)
	if err != nil {
		return append([]int{}, domain.DefaultWeekend...)
	}
	return days
}

// firstName — имя из ФИО («Фамилия Имя Отчество» → «Имя»).
func firstName(fio string) string {
	parts := splitWords(fio)
	if len(parts) > 1 {
		return parts[1]
	}
	if fio != "" {
		return fio
	}
	return "коллега"
}

func splitWords(s string) []string {
	var out []string
	word := ""
	for _, r := range s {
		if r == ' ' || r == '\t' {
			if word != "" {
				out = append(out, word)
				word = ""
			}
			continue
		}
		word += string(r)
	}
	if word != "" {
		out = append(out, word)
	}
	return out
}

// plural — русские формы: plural(n, "день", "дня", "дней").
func plural(n int, one, few, many string) string {
	if n < 0 {
		n = -n
	}
	if n%10 == 1 && n%100 != 11 {
		return one
	}
	if n%10 >= 2 && n%10 <= 4 && !(n%100 >= 12 && n%100 <= 14) {
		return few
	}
	return many
}

// userRoom — личная Socket.IO-комната пользователя.
func userRoom(userID int64) string {
	return "user_" + strconv.FormatInt(userID, 10)
}

func strconvI64(n int64) string { return strconv.FormatInt(n, 10) }
func strconvInt(n int) string   { return strconv.Itoa(n) }

func formatHours(h float64) string { return strconv.FormatFloat(h, 'f', -1, 64) }

func randIntn(n int) int { return rand.IntN(n) }
func randFloat() float64 { return rand.Float64() }

// truncateRunes — обрезка по рунам (тексты русские, байтовый срез порвал бы
// многобайтовый символ).
func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}
