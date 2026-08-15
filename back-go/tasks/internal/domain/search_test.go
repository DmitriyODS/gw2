package domain

import "testing"

// Регулярное выражение включается только тогда, когда человек его написал:
// обычная фраза не должна внезапно становиться шаблоном.
func TestSearchRegexDetection(t *testing.T) {
	cases := []struct {
		query string
		want  bool
	}{
		{"отчёт", false},
		{"т.к. сроки", false}, // точка — часть обычного текста
		{"отчёт|акт", true},
		{`^Заказ \d+`, true},
		{"счёт (копия)", true},
		{"сломанная скобка (", false}, // не компилируется — ищем подстрокой
		{"", false},
	}
	for _, c := range cases {
		if _, ok := SearchRegex(c.query); ok != c.want {
			t.Errorf("SearchRegex(%q) = %v, ожидалось %v", c.query, ok, c.want)
		}
	}
}

// Длинное выражение движку Postgres не отдаём: он с бэктрекингом, и «(a+)+b»
// на длинной строке занимает соединение целиком. Ищем подстрокой.
func TestSearchRegexLengthLimit(t *testing.T) {
	long := "(a+)+b"
	for len(long) <= maxRegexLen {
		long += "(a+)+b"
	}
	if _, ok := SearchRegex(long); ok {
		t.Errorf("SearchRegex(len=%d) принял слишком длинное выражение", len(long))
	}
	if _, ok := SearchRegex("отчёт|акт"); !ok {
		t.Error("обычное короткое выражение должно остаться регулярным")
	}
}

// Служебные символы подстрочного поиска экранируются: «100%» — это про
// проценты, а не «100 и что угодно».
func TestEscapeLike(t *testing.T) {
	if got := EscapeLike("100%"); got != `100\%` {
		t.Errorf("EscapeLike(100%%) = %q", got)
	}
	if got := EscapeLike("файл_1"); got != `файл\_1` {
		t.Errorf("EscapeLike(файл_1) = %q", got)
	}
}
