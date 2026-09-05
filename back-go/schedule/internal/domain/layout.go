package domain

import "sort"

/* Раскладка дня: занятость, окна и накладки.

   Считает это обычно клиент (front/src/utils/scheduleLayout.js) — он же и
   рисует. Здесь ровно столько же логики, сколько нужно живой плитке рабочего
   стола: «сколько сегодня занято» и «сколько времени в окнах». Правила у них
   общие, поэтому цифра на плитке совпадает с цифрой в сводке дня. */

// Span — отрезок времени в минутах от полуночи.
type Span struct {
	Start int
	End   int
}

// MergeSpans — склеить пересекающиеся отрезки. Накладка двух занятий не должна
// считаться двойной занятостью: в сутках от неё больше времени не становится.
func MergeSpans(spans []Span) []Span {
	if len(spans) == 0 {
		return nil
	}
	sorted := make([]Span, len(spans))
	copy(sorted, spans)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Start != sorted[j].Start {
			return sorted[i].Start < sorted[j].Start
		}
		return sorted[i].End < sorted[j].End
	})
	out := []Span{sorted[0]}
	for _, s := range sorted[1:] {
		last := &out[len(out)-1]
		if s.Start <= last.End {
			if s.End > last.End {
				last.End = s.End
			}
			continue
		}
		out = append(out, s)
	}
	return out
}

// BusyMin — сколько минут занято (без двойного счёта накладок).
func BusyMin(spans []Span) int {
	total := 0
	for _, s := range MergeSpans(spans) {
		total += s.End - s.Start
	}
	return total
}

// Gaps — свободные окна между занятиями длиной не меньше minGap. До первого
// занятия и после последнего окон нет: это ещё не день и уже не день.
func Gaps(spans []Span, minGap int) []Span {
	merged := MergeSpans(spans)
	out := []Span{}
	for i := 1; i < len(merged); i++ {
		gap := Span{Start: merged[i-1].End, End: merged[i].Start}
		if gap.End-gap.Start >= minGap {
			out = append(out, gap)
		}
	}
	return out
}

// GapMinutes — сколько всего времени приходится на окна.
func GapMinutes(spans []Span, minGap int) int {
	total := 0
	for _, g := range Gaps(spans, minGap) {
		total += g.End - g.Start
	}
	return total
}
