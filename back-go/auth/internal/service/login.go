package service

import (
	"context"
	"strconv"
	"strings"
)

// Генерация логина из ФИО: транслит фамилии (6 букв) + точка + первая буква
// имени + первая буква отчества. «Осиповский Дмитрий Сергеевич» → osipov.ds.
// Нет отчества → osipov.d; фамилия короче 6 букв → берётся целиком.

var translitMap = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "e",
	'ж': "zh", 'з': "z", 'и': "i", 'й': "j", 'к': "k", 'л': "l", 'м': "m",
	'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
	'ф': "f", 'х': "h", 'ц': "c", 'ч': "ch", 'ш': "sh", 'щ': "sch", 'ъ': "",
	'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
}

// translit — фонетическая транслитерация в латиницу [a-z0-9]; прочее отбрасывается.
func translit(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if mapped, ok := translitMap[r]; ok {
			b.WriteString(mapped)
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// initial — первая латинская буква транслита слова (одна буква имени/отчества).
func initial(word string) string {
	t := translit(word)
	if t == "" {
		return ""
	}
	return t[:1]
}

// genLogin — кандидат логина из ФИО (без проверки уникальности).
func genLogin(fio string) string {
	parts := strings.Fields(fio)
	if len(parts) == 0 {
		return ""
	}
	surname := translit(parts[0])
	if len(surname) > 6 {
		surname = surname[:6]
	}
	login := surname
	if len(parts) >= 2 {
		login += "." + initial(parts[1])
	}
	if len(parts) >= 3 {
		login += initial(parts[2])
	}
	return strings.Trim(login, ".")
}

// SuggestLogin — свободный логин-кандидат из ФИО для live-подсказки на фронте.
// При коллизии добавляет числовой суффикс. Гарантирует длину ≥3.
func (s *Service) SuggestLogin(ctx context.Context, fio string) (string, error) {
	base := genLogin(fio)
	if len(base) < 3 {
		base = "user"
	}
	for i := 0; i < 100; i++ {
		candidate := base
		if i > 0 {
			candidate = base + strconv.Itoa(i+1)
		}
		existing, err := s.repo.GetByLogin(ctx, candidate)
		if err != nil {
			return "", err
		}
		if existing == nil {
			return candidate, nil
		}
	}
	return base, nil
}
