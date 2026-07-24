// Package service — бизнес-логика питомцев-грувиков на портах домена:
// экономика (XP/кудосы), эволюция, магазин, прогулка/лечение/поглаживание,
// рейтинг признания и приватная история активности.
package service

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
)

type Service struct {
	pets         domain.PetRepo
	shop         domain.ShopRepo
	activity     domain.ActivityRepo
	bank         domain.BankRepo
	installments domain.InstallmentRepo
	users        domain.UserReader
	companies    domain.CompanyReader
	work         domain.WorkReader
	daily        domain.Daily
	pub          domain.EventPublisher
	log          *slog.Logger
}

func New(pets domain.PetRepo, shop domain.ShopRepo, activity domain.ActivityRepo,
	bank domain.BankRepo, installments domain.InstallmentRepo, users domain.UserReader,
	companies domain.CompanyReader, work domain.WorkReader, daily domain.Daily,
	pub domain.EventPublisher, log *slog.Logger) *Service {

	return &Service{
		pets: pets, shop: shop, activity: activity, bank: bank, installments: installments,
		users: users, companies: companies, work: work, daily: daily, pub: pub, log: log,
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

// grooveEnabled — включён ли режим «Мой Groove» у компании. Любая ошибка
// чтения → true (fail-open: режим по умолчанию включён).
func (s *Service) grooveEnabled(ctx context.Context, companyID int64) bool {
	ok, err := s.companies.GrooveEnabled(ctx, companyID)
	if err != nil {
		return true
	}
	return ok
}

// appendActivity — приватная история активности питомца; никогда не
// бросает (вызывающий не должен падать из-за журнала).
func (s *Service) appendActivity(ctx context.Context, petUserID int64, kind string, payload map[string]any) {
	if payload == nil {
		payload = map[string]any{}
	}
	if err := s.activity.Append(ctx, petUserID, kind, payload); err != nil {
		s.log.Warn("pets.activity_log_failed", "pet_user_id", petUserID, "kind", kind, "error", err)
	}
}

// appendLedger — запись в выписку кудо-банка; журнальная (fire-and-forget),
// как appendActivity: механика не должна падать из-за выписки. Банковские
// операции (переводы/вклад/кредит) пишут леджер сами — транзакционно.
func (s *Service) appendLedger(ctx context.Context, userID, companyID int64,
	delta int, kind string, counterpartyID *int64, comment string) {

	if delta == 0 {
		return
	}
	err := s.bank.AppendLedger(ctx, &domain.LedgerEntry{
		UserID: userID, CompanyID: companyID, Delta: delta, Kind: kind,
		CounterpartyID: counterpartyID, Comment: comment,
	})
	if err != nil {
		s.log.Warn("pets.ledger_append_failed", "user_id", userID, "kind", kind, "error", err)
	}
}

func userRoom(userID int64) string {
	return "user_" + strconv.FormatInt(userID, 10)
}

func strconvI64(n int64) string { return strconv.FormatInt(n, 10) }

func randIntn(n int) int { return rand.IntN(n) }

func containsStr(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
