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
