package domain

import (
	"testing"
	"time"
)

func msk() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.FixedZone("MSK", 3*60*60)
	}
	return loc
}

func at(loc *time.Location, y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, loc)
}

func TestNoRepeatHasNoNext(t *testing.T) {
	r := &Reminder{RemindAt: time.Now(), Repeat: Repeat{Kind: RepeatNone}}
	if _, ok := r.NextFire(time.Now()); ok {
		t.Fatal("разовое напоминание не должно повторяться")
	}
}

func TestDailyKeepsLocalTime(t *testing.T) {
	loc := msk()
	r := &Reminder{
		RemindAt: at(loc, 2026, time.July, 27, 9, 0), Timezone: "Europe/Moscow",
		Repeat: Repeat{Kind: RepeatDaily, Interval: 1},
	}
	next, ok := r.NextFire(at(loc, 2026, time.July, 27, 9, 0))
	if !ok {
		t.Fatal("ожидался следующий срок")
	}
	got := next.In(loc)
	if got.Day() != 28 || got.Hour() != 9 || got.Minute() != 0 {
		t.Fatalf("ожидалось 28-е 9:00 по месту, получено %s", got)
	}
}

func TestDailyCatchesUpAfterDowntime(t *testing.T) {
	loc := msk()
	r := &Reminder{
		RemindAt: at(loc, 2026, time.July, 1, 9, 0), Timezone: "Europe/Moscow",
		Repeat: Repeat{Kind: RepeatDaily, Interval: 1},
	}
	// Сервис лежал неделю: следующий срок — первый ПОСЛЕ текущего момента,
	// а не пачка пропущенных.
	next, ok := r.NextFire(at(loc, 2026, time.July, 8, 12, 0))
	if !ok {
		t.Fatal("ожидался следующий срок")
	}
	if got := next.In(loc); got.Day() != 9 || got.Hour() != 9 {
		t.Fatalf("ожидалось 9-е 9:00, получено %s", got)
	}
}

func TestWeekdaysSkipWeekend(t *testing.T) {
	loc := msk()
	// 31 июля 2026 — пятница.
	r := &Reminder{
		RemindAt: at(loc, 2026, time.July, 31, 10, 0), Timezone: "Europe/Moscow",
		Repeat: Repeat{Kind: RepeatWeekdays, Interval: 1},
	}
	next, ok := r.NextFire(at(loc, 2026, time.July, 31, 10, 0))
	if !ok {
		t.Fatal("ожидался следующий рабочий день")
	}
	if got := next.In(loc); got.Weekday() != time.Monday || got.Day() != 3 {
		t.Fatalf("ожидался понедельник 3 августа, получено %s", got)
	}
}

func TestWeeklySelectedDays(t *testing.T) {
	loc := msk()
	// 27 июля 2026 — понедельник; повтор по пн и чт.
	r := &Reminder{
		RemindAt: at(loc, 2026, time.July, 27, 8, 30), Timezone: "Europe/Moscow",
		Repeat: Repeat{Kind: RepeatWeekly, Interval: 1, Days: []int{1, 4}},
	}
	next, ok := r.NextFire(at(loc, 2026, time.July, 27, 8, 30))
	if !ok {
		t.Fatal("ожидался следующий срок")
	}
	if got := next.In(loc); got.Weekday() != time.Thursday || got.Hour() != 8 || got.Minute() != 30 {
		t.Fatalf("ожидался четверг 8:30, получено %s", got)
	}
}

func TestMonthlyClampsShortMonth(t *testing.T) {
	loc := msk()
	r := &Reminder{
		RemindAt: at(loc, 2026, time.January, 31, 12, 0), Timezone: "Europe/Moscow",
		Repeat: Repeat{Kind: RepeatMonthly, Interval: 1},
	}
	next, ok := r.NextFire(at(loc, 2026, time.January, 31, 12, 0))
	if !ok {
		t.Fatal("ожидался следующий срок")
	}
	if got := next.In(loc); got.Month() != time.February || got.Day() != 28 {
		t.Fatalf("ожидалось 28 февраля, получено %s", got)
	}
}

func TestUntilStopsRepeat(t *testing.T) {
	loc := msk()
	until := at(loc, 2026, time.July, 28, 0, 0)
	r := &Reminder{
		RemindAt: at(loc, 2026, time.July, 27, 9, 0), Timezone: "Europe/Moscow",
		Repeat: Repeat{Kind: RepeatDaily, Interval: 1, Until: &until},
	}
	if _, ok := r.NextFire(at(loc, 2026, time.July, 27, 9, 0)); ok {
		t.Fatal("после Until повторов быть не должно")
	}
}
