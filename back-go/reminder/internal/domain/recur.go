package domain

import "time"

// maxRecurSteps — предохранитель шага повтора: при странных данных (например,
// weekly без валидных дней) цикл обязан завершиться, а не крутиться вечно.
const maxRecurSteps = 400

// Location — зона пользователя; неизвестная/пустая — UTC (напоминание всё
// равно сработает вовремя, «поплывёт» только привязка к местным часам).
func (r *Reminder) Location() *time.Location {
	if r.Timezone == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(r.Timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

// NextFire — время следующего срабатывания строго после after; ok=false, когда
// повторов больше нет (разовое напоминание или вышли за Repeat.Until).
//
// Шаг считается в локальной зоне пользователя: «каждый день в 9:00» остаётся
// девятью утра и после перевода часов, а «каждое 31-е» в коротком месяце
// приходится на его последний день (как в календарях телефонов).
func (r *Reminder) NextFire(after time.Time) (time.Time, bool) {
	if r.Repeat.Kind == "" || r.Repeat.Kind == RepeatNone {
		return time.Time{}, false
	}
	loc := r.Location()
	step := r.Repeat.Interval
	if step < 1 {
		step = 1
	}
	cur := r.RemindAt.In(loc)
	after = after.In(loc)

	for i := 0; i < maxRecurSteps; i++ {
		next, ok := advance(cur, r.Repeat, step, loc)
		if !ok {
			return time.Time{}, false
		}
		cur = next
		if cur.After(after) {
			if r.Repeat.Until != nil && cur.After(r.Repeat.Until.In(loc)) {
				return time.Time{}, false
			}
			return cur.UTC(), true
		}
	}
	return time.Time{}, false
}

// advance — один шаг правила от текущего времени.
func advance(cur time.Time, rep Repeat, step int, loc *time.Location) (time.Time, bool) {
	switch rep.Kind {
	case RepeatDaily:
		return shiftDays(cur, step, loc), true
	case RepeatWeekdays:
		next := shiftDays(cur, 1, loc)
		for next.Weekday() == time.Saturday || next.Weekday() == time.Sunday {
			next = shiftDays(next, 1, loc)
		}
		return next, true
	case RepeatWeekly:
		return nextWeekly(cur, rep, step, loc), true
	case RepeatMonthly:
		return shiftMonths(cur, step, loc), true
	case RepeatYearly:
		return shiftMonths(cur, 12*step, loc), true
	}
	return time.Time{}, false
}

// nextWeekly — следующий из выбранных дней недели; пустой список дней означает
// «в тот же день недели каждые N недель».
func nextWeekly(cur time.Time, rep Repeat, step int, loc *time.Location) time.Time {
	days := weekdaySet(rep.Days)
	if len(days) == 0 {
		return shiftDays(cur, 7*step, loc)
	}
	// Внутри недели переходим к ближайшему следующему выбранному дню, а
	// перевалив воскресенье — прыгаем на step недель вперёд, к первому дню.
	for i := 1; i <= 7; i++ {
		cand := shiftDays(cur, i, loc)
		if days[isoWeekday(cand.Weekday())] {
			if step > 1 && cand.After(endOfWeek(cur, loc)) {
				return shiftDays(cand, 7*(step-1), loc)
			}
			return cand
		}
	}
	return shiftDays(cur, 7*step, loc)
}

// weekdaySet — множество дней недели 1..7 (1 — понедельник).
func weekdaySet(days []int) map[int]bool {
	set := map[int]bool{}
	for _, d := range days {
		if d >= 1 && d <= 7 {
			set[d] = true
		}
	}
	return set
}

func isoWeekday(w time.Weekday) int {
	if w == time.Sunday {
		return 7
	}
	return int(w)
}

// endOfWeek — воскресенье текущей недели (граница «перевалили неделю»).
func endOfWeek(t time.Time, loc *time.Location) time.Time {
	return shiftDays(t, 7-isoWeekday(t.Weekday()), loc)
}

// shiftDays — сдвиг на n суток С СОХРАНЕНИЕМ местного времени суток (перевод
// часов не должен смещать «9:00» на 8:00 или 10:00).
func shiftDays(t time.Time, n int, loc *time.Location) time.Time {
	t = t.In(loc)
	y, m, d := t.Date()
	return time.Date(y, m, d+n, t.Hour(), t.Minute(), t.Second(), 0, loc)
}

// shiftMonths — сдвиг на n месяцев; если в целевом месяце нет такого числа
// (31-е в феврале), берётся последний день месяца.
func shiftMonths(t time.Time, n int, loc *time.Location) time.Time {
	t = t.In(loc)
	y, m, d := t.Date()
	target := time.Date(y, m+time.Month(n), 1, t.Hour(), t.Minute(), t.Second(), 0, loc)
	if last := daysInMonth(target); d > last {
		d = last
	}
	ty, tm, _ := target.Date()
	return time.Date(ty, tm, d, t.Hour(), t.Minute(), t.Second(), 0, loc)
}

func daysInMonth(t time.Time) int {
	y, m, _ := t.Date()
	return time.Date(y, m+1, 0, 0, 0, 0, 0, t.Location()).Day()
}
