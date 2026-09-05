package service

import (
	"context"
	"time"

	"github.com/DmitriyODS/gw2/back-go/schedule/internal/domain"
)

/* Живая плитка рабочего стола и глобальный поиск Hola: и то и другое идёт
   сразу по ВСЕМ доступным расписаниям — своим и открытым адресно.

   День и текущую минуту присылает клиент: зон у расписаний много, а «сейчас»
   у человека одно, и считать его сервер должен по тому времени, которое видит
   человек (тот же приём, что у повестки ежедневников). */

// Agenda — что идёт сейчас, что дальше и сколько сегодня занято.
func (s *Service) Agenda(ctx context.Context, userID int64, date time.Time, minute int) (*domain.Agenda, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	weekday := (int(date.Weekday()) + 6) % 7
	rows, err := s.repo.ItemsForDay(ctx, a.UserID, a.Companies, weekday)
	if err != nil {
		return nil, err
	}

	out := &domain.Agenda{}
	spans := make([]domain.Span, 0, len(rows))
	// Порог окна берём самый мягкий из расписаний дня: на плитке сводка общая,
	// и прятать окно только потому, что в одном из расписаний порог крупнее,
	// было бы неверно.
	minGap := 0
	for _, row := range rows {
		if !row.Item.OccursOn(row.Schedule, date) {
			continue
		}
		out.Total++
		spans = append(spans, domain.Span{Start: row.Item.StartMin, End: row.Item.EndMin})
		if minGap == 0 || row.Schedule.GapMin < minGap {
			minGap = row.Schedule.GapMin
		}
		entry := &domain.AgendaItem{
			ScheduleID: row.Schedule.ID, ScheduleName: row.Schedule.Name,
			ItemID: row.Item.ID, Title: row.Item.Title,
			StartMin: row.Item.StartMin, EndMin: row.Item.EndMin, Color: row.Color,
		}
		switch {
		case row.Item.StartMin <= minute && minute < row.Item.EndMin:
			if out.Now == nil || row.Item.EndMin > out.Now.EndMin {
				out.Now = entry
			}
		case row.Item.StartMin > minute:
			if out.Next == nil || row.Item.StartMin < out.Next.StartMin {
				out.Next = entry
			}
		}
	}
	if minGap == 0 {
		minGap = 20
	}
	out.BusyMin = domain.BusyMin(spans)
	out.GapMin = domain.GapMinutes(spans, minGap)
	return out, nil
}

// Search — занятия всех доступных расписаний по строке поиска.
func (s *Service) Search(ctx context.Context, userID int64, query string, limit int) ([]*domain.ItemScope, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if query == "" {
		return []*domain.ItemScope{}, nil
	}
	return s.repo.SearchItems(ctx, a.UserID, a.Companies, query, limit)
}
