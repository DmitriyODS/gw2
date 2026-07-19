// Разбор дат в русских фразах: «на завтра», «в пятницу», «на 15 июля».
package service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const dayLayout = "2006-01-02"

var months = map[string]time.Month{
	"января": 1, "февраля": 2, "марта": 3, "апреля": 4, "мая": 5, "июня": 6,
	"июля": 7, "августа": 8, "сентября": 9, "октября": 10, "ноября": 11, "декабря": 12,
}

var weekdays = map[string]time.Weekday{
	"понедельник": time.Monday, "вторник": time.Tuesday,
	"среду": time.Wednesday, "среда": time.Wednesday,
	"четверг": time.Thursday,
	"пятницу": time.Friday, "пятница": time.Friday,
	"субботу": time.Saturday, "суббота": time.Saturday,
	"воскресенье": time.Sunday,
}

var (
	reRelDay  = regexp.MustCompile(`(?:^|\s)(?:на |в )?(сегодня|завтра|послезавтра)(?:\s|$)`)
	reWeekday = regexp.MustCompile(`(?:^|\s)(?:на |в |во )(понедельник|вторник|среду|среда|четверг|пятницу|пятница|субботу|суббота|воскресенье)(?:\s|$)`)
	reDayMon  = regexp.MustCompile(`(?:^|\s)(?:на )?(\d{1,2}) (января|февраля|марта|апреля|мая|июня|июля|августа|сентября|октября|ноября|декабря)(?:\s|$)`)
)

// ExtractDate — найти дату во фразе, вернуть её (YYYY-MM-DD в зоне now) и
// фразу без датового фрагмента. Ближайший день недели — вперёд (сегодняшний
// день недели означает сегодня).
func ExtractDate(s string, now time.Time) (date, cleaned string, found bool) {
	cut := func(m []int) string {
		return strings.Join(strings.Fields(s[:m[0]]+" "+s[m[1]:]), " ")
	}
	if m := reRelDay.FindStringSubmatchIndex(s); m != nil {
		word := s[m[2]:m[3]]
		add := map[string]int{"сегодня": 0, "завтра": 1, "послезавтра": 2}[word]
		return now.AddDate(0, 0, add).Format(dayLayout), cut(m), true
	}
	if m := reWeekday.FindStringSubmatchIndex(s); m != nil {
		wd := weekdays[s[m[2]:m[3]]]
		days := (int(wd) - int(now.Weekday()) + 7) % 7
		return now.AddDate(0, 0, days).Format(dayLayout), cut(m), true
	}
	if m := reDayMon.FindStringSubmatchIndex(s); m != nil {
		day, _ := strconv.Atoi(s[m[2]:m[3]])
		mon := months[s[m[4]:m[5]]]
		year := now.Year()
		d := time.Date(year, mon, day, 0, 0, 0, 0, now.Location())
		// Прошедшая в этом году дата означает следующий год.
		if d.Before(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())) {
			d = d.AddDate(1, 0, 0)
		}
		return d.Format(dayLayout), cut(m), true
	}
	return "", strings.TrimSpace(s), false
}

// HumanDate — дата для реплики: «сегодня», «завтра» либо «15 июля».
func HumanDate(dateISO string, now time.Time) string {
	d, err := time.ParseInLocation(dayLayout, dateISO, now.Location())
	if err != nil {
		return dateISO
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch int(d.Sub(today).Hours() / 24) {
	case 0:
		return "сегодня"
	case 1:
		return "завтра"
	case 2:
		return "послезавтра"
	}
	var monthGen = [...]string{"января", "февраля", "марта", "апреля", "мая", "июня",
		"июля", "августа", "сентября", "октября", "ноября", "декабря"}
	return fmt.Sprintf("%d %s", d.Day(), monthGen[d.Month()-1])
}
