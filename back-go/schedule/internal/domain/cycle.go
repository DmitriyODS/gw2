package domain

import (
	"sort"
	"time"
)

/* Цикличность расписания.

   Расписание регулярное: вхождений на конкретные даты не существует, и «идёт
   ли занятие на этой неделе» считается из даты каждый раз. Способов сказать
   это два, и они взаимоисключающие:

     1. Номера недель цикла (Item.Weeks) — прямое обобщение числителя и
        знаменателя: цикл расписания N недель, занятие идёт в перечисленных.
        Пустой набор означает «каждую неделю».
     2. Своё правило (Item.RepeatEvery/RepeatFrom/RepeatUntil) — «каждые N
        недель с такой-то даты», живёт мимо цикла расписания.

   ИНВАРИАНТ: правило считают двое — этот файл (живая плитка, выгрузка, поиск)
   и front/src/utils/scheduleCycle.js (экран). Меняете здесь — меняйте там же,
   тестовые кейсы у них общие. */

// MaxCycleWeeks — потолок длины цикла. Длиннее человек уже не удерживает в
// голове, а названия недель превращаются в простыню.
const MaxCycleWeeks = 4

const weekSpan = 7 * 24 * time.Hour

// MondayOf — понедельник недели, в которую попадает дата (UTC-полночь).
// Все расчёты цикла идут по UTC-датам: перевод часов не должен сдвигать
// границу недели.
func MondayOf(t time.Time) time.Time {
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
}

// weeksBetween — целых недель между двумя понедельниками (может быть < 0).
func weeksBetween(from, to time.Time) int { return int(to.Sub(from) / weekSpan) }

// WeekIndex — номер недели цикла (1..cycleWeeks) для недели с понедельником
// monday. Считается в обе стороны от якоря: прошлые недели тоже нумеруются.
func WeekIndex(anchor, monday time.Time, cycleWeeks int) int {
	if cycleWeeks < 1 {
		cycleWeeks = 1
	}
	diff := weeksBetween(MondayOf(anchor), MondayOf(monday))
	return ((diff%cycleWeeks)+cycleWeeks)%cycleWeeks + 1
}

// WeekIndexOf — номер недели цикла расписания для даты.
func (s *Schedule) WeekIndexOf(date time.Time) int {
	return WeekIndex(s.CycleAnchor, date, s.CycleWeeks)
}

// WeekLabel — название недели цикла (idx с единицы): своё, если задано,
// иначе пусто — подпись «Неделя N» рисует клиент.
func (s *Schedule) WeekLabel(idx int) string {
	if idx < 1 || idx > len(s.WeekLabels) {
		return ""
	}
	return s.WeekLabels[idx-1]
}

// OccursOn — идёт ли занятие на неделе с понедельником monday.
func (i *Item) OccursOn(s *Schedule, monday time.Time) bool {
	monday = MondayOf(monday)
	if i.RepeatEvery != nil && *i.RepeatEvery > 0 {
		if i.RepeatFrom == nil {
			return false
		}
		from := MondayOf(*i.RepeatFrom)
		if monday.Before(from) {
			return false
		}
		if i.RepeatUntil != nil && monday.After(MondayOf(*i.RepeatUntil)) {
			return false
		}
		return weeksBetween(from, monday)%*i.RepeatEvery == 0
	}
	if len(i.Weeks) == 0 {
		return true
	}
	idx := WeekIndex(s.CycleAnchor, monday, s.CycleWeeks)
	for _, w := range i.Weeks {
		if w == idx {
			return true
		}
	}
	return false
}

// OccursAt — идёт ли занятие в конкретный день (совпал день недели и неделя).
func (i *Item) OccursAt(s *Schedule, date time.Time) bool {
	if (int(date.Weekday())+6)%7 != i.Weekday {
		return false
	}
	return i.OccursOn(s, date)
}

// NormalizeWeeks — привести набор недель к циклу: выбросить вышедшие за его
// длину, убрать дубли, упорядочить. Полный набор сворачивается в пустой —
// «во все недели цикла» и «каждую неделю» это одно и то же, и хранить их
// по-разному значило бы получить занятие, пропадающее при смене длины цикла.
// Отсюда же следует поведение при УКОРОЧЕНИИ цикла: занятие, у которого не
// осталось ни одной действительной недели, становится еженедельным — молча
// исчезнуть со шкалы оно не должно.
func NormalizeWeeks(weeks []int, cycleWeeks int) []int {
	if cycleWeeks < 1 {
		cycleWeeks = 1
	}
	seen := make(map[int]bool, len(weeks))
	out := make([]int, 0, len(weeks))
	for _, w := range weeks {
		if w < 1 || w > cycleWeeks || seen[w] {
			continue
		}
		seen[w] = true
		out = append(out, w)
	}
	if len(out) == 0 || len(out) == cycleWeeks {
		return []int{}
	}
	sort.Ints(out)
	return out
}

// NormalizeLabels — названия недель под длину цикла: лишние отбрасываются,
// недостающие добираются пустыми (клиент подпишет их «Неделя N»).
func NormalizeLabels(labels []string, cycleWeeks int) []string {
	if cycleWeeks < 1 {
		cycleWeeks = 1
	}
	out := make([]string, cycleWeeks)
	for i := range out {
		if i < len(labels) {
			out[i] = trimLabel(labels[i])
		}
	}
	return out
}

func trimLabel(s string) string {
	const max = 40
	r := []rune(s)
	if len(r) > max {
		return string(r[:max])
	}
	return string(r)
}

// NormalizeCycleWeeks — длина цикла в допустимых границах.
func NormalizeCycleWeeks(n int) int {
	if n < 1 {
		return 1
	}
	if n > MaxCycleWeeks {
		return MaxCycleWeeks
	}
	return n
}

// NormalizeGap — порог окна: слишком мелкий превращает каждую перемену в
// «окно», слишком крупный прячет их все.
func NormalizeGap(n int) int {
	switch {
	case n < 5:
		return 5
	case n > 240:
		return 240
	default:
		return n
	}
}
