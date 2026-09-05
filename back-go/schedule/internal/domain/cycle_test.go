package domain

import (
	"testing"
	"time"
)

// day — дата YYYY-MM-DD в UTC (в тестах читается лучше, чем time.Date).
func day(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse(DateLayout, s)
	if err != nil {
		t.Fatalf("плохая дата %q: %v", s, err)
	}
	return d
}

func TestMondayOf(t *testing.T) {
	// 2026-09-05 — суббота; понедельник её недели — 31 августа.
	cases := map[string]string{
		"2026-08-31": "2026-08-31", // сам понедельник
		"2026-09-05": "2026-08-31",
		"2026-09-06": "2026-08-31", // воскресенье принадлежит своей неделе
		"2026-09-07": "2026-09-07",
	}
	for in, want := range cases {
		if got := MondayOf(day(t, in)).Format(DateLayout); got != want {
			t.Errorf("MondayOf(%s) = %s, ожидалось %s", in, got, want)
		}
	}
}

func TestWeekIndex(t *testing.T) {
	anchor := day(t, "2026-08-31") // неделя №1

	// Цикл из двух недель — числитель и знаменатель.
	cases := []struct {
		date  string
		cycle int
		want  int
	}{
		{"2026-08-31", 2, 1},
		{"2026-09-05", 2, 1}, // суббота той же недели
		{"2026-09-07", 2, 2},
		{"2026-09-14", 2, 1},
		// Прошлое нумеруется в обе стороны от якоря.
		{"2026-08-24", 2, 2},
		{"2026-08-17", 2, 1},
		// Цикл из четырёх недель.
		{"2026-08-31", 4, 1},
		{"2026-09-07", 4, 2},
		{"2026-09-21", 4, 4},
		{"2026-09-28", 4, 1},
		{"2026-08-24", 4, 4},
		// Цикл из одной недели: неделя всегда первая.
		{"2026-09-14", 1, 1},
	}
	for _, c := range cases {
		got := WeekIndex(anchor, day(t, c.date), c.cycle)
		if got != c.want {
			t.Errorf("WeekIndex(%s, цикл %d) = %d, ожидалось %d", c.date, c.cycle, got, c.want)
		}
	}
}

func TestOccursOnWeeks(t *testing.T) {
	sc := &Schedule{CycleWeeks: 2, CycleAnchor: day(t, "2026-08-31")}

	every := &Item{Weeks: []int{}}
	numerator := &Item{Weeks: []int{1}}
	denominator := &Item{Weeks: []int{2}}

	weekOne, weekTwo := day(t, "2026-08-31"), day(t, "2026-09-07")

	if !every.OccursOn(sc, weekOne) || !every.OccursOn(sc, weekTwo) {
		t.Error("пустой набор недель означает «каждую неделю»")
	}
	if !numerator.OccursOn(sc, weekOne) || numerator.OccursOn(sc, weekTwo) {
		t.Error("занятие недели 1 идёт только в числитель")
	}
	if denominator.OccursOn(sc, weekOne) || !denominator.OccursOn(sc, weekTwo) {
		t.Error("занятие недели 2 идёт только в знаменатель")
	}
}

func TestOccursOnOwnRepeat(t *testing.T) {
	sc := &Schedule{CycleWeeks: 2, CycleAnchor: day(t, "2026-08-31")}
	from := day(t, "2026-08-31")
	until := day(t, "2026-10-05")
	every := 3
	item := &Item{RepeatEvery: &every, RepeatFrom: &from, RepeatUntil: &until}

	cases := map[string]bool{
		"2026-08-24": false, // раньше начала
		"2026-08-31": true,
		"2026-09-07": false,
		"2026-09-14": false,
		"2026-09-21": true, // +3 недели
		"2026-10-05": false,
		"2026-10-12": false, // позже конца, хотя и кратно трём
	}
	for date, want := range cases {
		if got := item.OccursOn(sc, day(t, date)); got != want {
			t.Errorf("OccursOn(%s) = %v, ожидалось %v", date, got, want)
		}
	}

	// Своё правило не смотрит на цикл расписания: 12 октября кратно трём от
	// начала, но обрезано датой конца — а вот 5 октября не кратно.
	noEnd := &Item{RepeatEvery: &every, RepeatFrom: &from}
	if !noEnd.OccursOn(sc, day(t, "2026-10-12")) {
		t.Error("без даты конца повтор продолжается")
	}
}

func TestOccursAtChecksWeekday(t *testing.T) {
	sc := &Schedule{CycleWeeks: 1, CycleAnchor: day(t, "2026-08-31")}
	item := &Item{Weekday: 1} // вторник

	if !item.OccursAt(sc, day(t, "2026-09-01")) {
		t.Error("занятие вторника идёт во вторник")
	}
	if item.OccursAt(sc, day(t, "2026-09-02")) {
		t.Error("занятие вторника не идёт в среду")
	}
}

func TestNormalizeWeeks(t *testing.T) {
	cases := []struct {
		in    []int
		cycle int
		want  []int
	}{
		{[]int{1, 3}, 4, []int{1, 3}},
		{[]int{3, 1}, 4, []int{1, 3}},    // упорядочивается
		{[]int{1, 1, 2}, 4, []int{1, 2}}, // дубли убираются
		// Полный набор — это «каждую неделю», хранить его перечислением нельзя:
		// иначе при смене длины цикла занятие поредело бы само.
		{[]int{1, 2}, 2, []int{}},
		// Укорочение цикла: недели за его пределами отбрасываются, а занятие,
		// у которого не осталось ни одной, становится еженедельным.
		{[]int{3, 4}, 2, []int{}},
		{[]int{1, 3}, 2, []int{1}},
		{[]int{0, -1, 9}, 4, []int{}},
	}
	for _, c := range cases {
		got := NormalizeWeeks(c.in, c.cycle)
		if len(got) != len(c.want) {
			t.Fatalf("NormalizeWeeks(%v, %d) = %v, ожидалось %v", c.in, c.cycle, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("NormalizeWeeks(%v, %d) = %v, ожидалось %v", c.in, c.cycle, got, c.want)
			}
		}
	}
}

func TestNormalizeLabels(t *testing.T) {
	got := NormalizeLabels([]string{"Числитель", "Знаменатель", "Лишняя"}, 2)
	if len(got) != 2 || got[0] != "Числитель" || got[1] != "Знаменатель" {
		t.Errorf("лишние названия недель отбрасываются, получено %v", got)
	}
	got = NormalizeLabels([]string{"А"}, 3)
	if len(got) != 3 || got[0] != "А" || got[1] != "" || got[2] != "" {
		t.Errorf("недостающие названия добираются пустыми, получено %v", got)
	}
}

func TestNormalizeCycleWeeksAndGap(t *testing.T) {
	if NormalizeCycleWeeks(0) != 1 || NormalizeCycleWeeks(9) != MaxCycleWeeks {
		t.Error("длина цикла живёт в границах 1..MaxCycleWeeks")
	}
	if NormalizeGap(1) != 5 || NormalizeGap(1000) != 240 || NormalizeGap(35) != 35 {
		t.Error("порог окна живёт в границах 5..240")
	}
}

func TestItemNormalize(t *testing.T) {
	item := &Item{Title: "  Матан  ", Weekday: 1, StartMin: 600, EndMin: 700, Weeks: []int{1, 5}}
	if err := item.Normalize(2); err != nil {
		t.Fatalf("нормализация упала: %v", err)
	}
	if item.Title != "Матан" {
		t.Errorf("название обрезается по краям, получено %q", item.Title)
	}
	if len(item.Weeks) != 1 || item.Weeks[0] != 1 {
		t.Errorf("недели вне цикла отбрасываются, получено %v", item.Weeks)
	}

	if err := (&Item{Title: "x", EndMin: 10, StartMin: 20}).Normalize(1); err == nil {
		t.Error("конец занятия обязан быть позже начала")
	}
	if err := (&Item{Title: "", StartMin: 0, EndMin: 10}).Normalize(1); err == nil {
		t.Error("название обязательно")
	}

	// Своё правило повтора отменяет номера недель: два описания одного и того
	// же занятия противоречили бы друг другу.
	every := 2
	from := day(t, "2026-09-02") // среда — приводится к понедельнику
	own := &Item{Title: "x", StartMin: 0, EndMin: 10, Weeks: []int{1}, RepeatEvery: &every, RepeatFrom: &from}
	if err := own.Normalize(2); err != nil {
		t.Fatalf("нормализация своего правила упала: %v", err)
	}
	if len(own.Weeks) != 0 {
		t.Errorf("номера недель при своём правиле очищаются, получено %v", own.Weeks)
	}
	if own.RepeatFrom.Format(DateLayout) != "2026-08-31" {
		t.Errorf("начало своего правила приводится к понедельнику, получено %s",
			own.RepeatFrom.Format(DateLayout))
	}

	// Правило без даты начала невыполнимо.
	if err := (&Item{Title: "x", StartMin: 0, EndMin: 10, RepeatEvery: &every}).Normalize(1); err == nil {
		t.Error("своё правило требует дату начала")
	}
}
