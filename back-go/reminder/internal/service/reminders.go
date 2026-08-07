package service

import (
	"context"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/reminder/internal/domain"
)

const (
	maxTitleRunes = 300
	maxNoteRunes  = 2000
	// upcomingWindow — горизонт «ближайших» для живой плитки рабочего стола.
	upcomingWindow = 14 * 24 * time.Hour
	// snoozeMin/snoozeMax — границы «отложить» (минуты).
	snoozeMin = 1
	snoozeMax = 24 * 60
)

// List — напоминания пользователя: активные по возрастанию срока, сработавшие
// — журналом по убыванию.
func (s *Service) List(ctx context.Context, userID int64, scope domain.ListScope) ([]*domain.Reminder, error) {
	switch scope {
	case domain.ScopeDone, domain.ScopeAll:
	default:
		scope = domain.ScopeActive
	}
	return s.repo.List(ctx, userID, scope)
}

// Upcoming — ближайшие активные напоминания (живая плитка и центр уведомлений).
func (s *Service) Upcoming(ctx context.Context, userID int64, limit int) ([]*domain.Reminder, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	return s.repo.Upcoming(ctx, userID, time.Now().Add(upcomingWindow), limit)
}

// ByLink — напоминания, привязанные к записи ежедневника/календаря: раздел,
// который правит запись, обновляет ими свой снимок (время и название).
func (s *Service) ByLink(ctx context.Context, userID int64, kind string, recordID int64) ([]*domain.Reminder, error) {
	if !domain.LinkKinds[kind] || kind == domain.LinkNone {
		return nil, domain.ErrBadLink
	}
	return s.repo.ByLink(ctx, userID, kind, recordID)
}

func (s *Service) Get(ctx context.Context, userID, id int64) (*domain.Reminder, error) {
	return s.requireOwned(ctx, userID, id)
}

// Create — новое напоминание. Время приходит от клиента уже с учётом «за N
// минут до события» (привязки) — сервер хранит момент срабатывания как есть.
func (s *Service) Create(ctx context.Context, r *domain.Reminder) (*domain.Reminder, error) {
	if err := normalize(r); err != nil {
		return nil, err
	}
	r.Active = true
	if err := s.repo.Create(ctx, r); err != nil {
		return nil, err
	}
	s.publish(ctx, "reminder:created", r)
	return r, nil
}

// Update — частичная правка; смена срока/повтора возвращает напоминание в
// строй (сработавшее разовое, которому передвинули время, снова ждёт).
func (s *Service) Update(ctx context.Context, userID, id int64, u domain.ReminderUpdate) (*domain.Reminder, error) {
	r, err := s.requireOwned(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if u.Title != nil {
		r.Title = *u.Title
	}
	if u.Note != nil {
		r.Note = *u.Note
	}
	if u.Timezone != nil {
		r.Timezone = *u.Timezone
	}
	if u.Repeat != nil {
		r.Repeat = *u.Repeat
	}
	if u.Link != nil {
		r.Link = *u.Link
	}
	if u.RemindAt != nil {
		r.RemindAt = *u.RemindAt
		if u.Active == nil && r.RemindAt.After(time.Now()) {
			r.Active = true
		}
	}
	if u.Active != nil {
		r.Active = *u.Active
	}
	if err := normalize(r); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, r); err != nil {
		return nil, err
	}
	s.publish(ctx, "reminder:updated", r)
	return r, nil
}

func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	r, err := s.requireOwned(ctx, userID, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.bus.Publish(ctx, "reminder:deleted", []string{userRoom(userID)},
		map[string]any{"id": r.ID, "owner_id": userID})
	return nil
}

// Snooze — отложить на minutes минут от текущего момента (кнопка «Ещё раз
// через 10 минут» в уведомлении).
func (s *Service) Snooze(ctx context.Context, userID, id int64, minutes int) (*domain.Reminder, error) {
	if minutes < snoozeMin || minutes > snoozeMax {
		return nil, domain.ErrBadSnooze
	}
	r, err := s.requireOwned(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	r.RemindAt = time.Now().Add(time.Duration(minutes) * time.Minute).Truncate(time.Second)
	r.Active = true
	if err := s.repo.Update(ctx, r); err != nil {
		return nil, err
	}
	s.publish(ctx, "reminder:updated", r)
	return r, nil
}

// Complete — «сделано»: разовое уходит в журнал, повторяющееся перескакивает на
// следующий срок (текущее считается закрытым).
func (s *Service) Complete(ctx context.Context, userID, id int64) (*domain.Reminder, error) {
	r, err := s.requireOwned(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	r.LastFiredAt = &now
	if next, ok := r.NextFire(now); ok {
		r.RemindAt = next
		r.Active = true
	} else {
		r.Active = false
	}
	if err := s.repo.Update(ctx, r); err != nil {
		return nil, err
	}
	s.publish(ctx, "reminder:updated", r)
	return r, nil
}

// normalize — проверка и приведение полей (общая для создания и правки).
func normalize(r *domain.Reminder) error {
	r.Title = trimRunes(r.Title, maxTitleRunes)
	if r.Title == "" {
		return domain.ErrTitleRequired
	}
	r.Note = trimRunes(r.Note, maxNoteRunes)
	if r.RemindAt.IsZero() {
		return domain.ErrTimeRequired
	}
	r.RemindAt = r.RemindAt.Truncate(time.Second)

	if r.Repeat.Kind == "" {
		r.Repeat.Kind = domain.RepeatNone
	}
	if !domain.RepeatKinds[r.Repeat.Kind] {
		return domain.ErrBadRepeat
	}
	if r.Repeat.Interval < 1 {
		r.Repeat.Interval = 1
	}
	if r.Repeat.Interval > 365 {
		r.Repeat.Interval = 365
	}
	r.Repeat.Days = cleanDays(r.Repeat.Days)

	if r.Link.Kind == "" {
		r.Link.Kind = domain.LinkNone
	}
	if !domain.LinkKinds[r.Link.Kind] {
		return domain.ErrBadLink
	}
	if r.Link.Kind == domain.LinkNone {
		r.Link = domain.Link{Kind: domain.LinkNone}
	} else {
		r.Link.Title = trimRunes(r.Link.Title, maxTitleRunes)
		if r.Link.LeadMinutes < 0 {
			r.Link.LeadMinutes = 0
		}
	}
	if r.Timezone == "" {
		r.Timezone = "UTC"
	}
	return nil
}

// cleanDays — дни недели 1..7 без дублей, по возрастанию.
func cleanDays(days []int) []int {
	seen := map[int]bool{}
	out := []int{}
	for d := 1; d <= 7; d++ {
		for _, v := range days {
			if v == d && !seen[d] {
				seen[d] = true
				out = append(out, d)
			}
		}
	}
	return out
}

func trimRunes(s string, max int) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > max {
		return strings.TrimSpace(string(r[:max]))
	}
	return s
}
