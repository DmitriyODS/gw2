package domain

import "testing"

// Проверка значений — единственная защита формы: ответ приходит и от участника,
// и от гостя по ссылке, и клиенту верить нельзя.

func TestCoerceAnswerChoice(t *testing.T) {
	q := Question{
		ID: 1, Type: QRadio, Title: "Цвет",
		Config: map[string]any{"options": []any{"Красный", "Синий"}},
	}

	if v, err := q.CoerceAnswer("Синий"); err != nil || v != "Синий" {
		t.Fatalf("законный вариант отвергнут: %v, %v", v, err)
	}
	if _, err := q.CoerceAnswer("Зелёный"); err == nil {
		t.Fatal("вариант вне списка должен отвергаться")
	}

	// «Другое» открывает свой текст — но только когда автор его включил.
	q.Config["other"] = true
	if v, err := q.CoerceAnswer("Зелёный"); err != nil || v != "Зелёный" {
		t.Fatalf("свой вариант при включённом «Другое»: %v, %v", v, err)
	}
}

func TestCoerceAnswerCheckboxLimits(t *testing.T) {
	q := Question{
		ID: 2, Type: QCheckbox, Title: "Языки",
		Config: map[string]any{
			"options": []any{"Go", "Vue", "SQL"}, "min_choices": 2, "max_choices": 2,
		},
	}

	if _, err := q.CoerceAnswer([]any{"Go"}); err == nil {
		t.Fatal("выбор меньше минимума должен отвергаться")
	}
	if _, err := q.CoerceAnswer([]any{"Go", "Vue", "SQL"}); err == nil {
		t.Fatal("выбор больше максимума должен отвергаться")
	}
	v, err := q.CoerceAnswer([]any{"Go", "Vue", "Go"})
	if err != nil {
		t.Fatalf("законный выбор отвергнут: %v", err)
	}
	// Дубли схлопываются: два одинаковых флажка — это один ответ.
	if list, ok := v.([]any); !ok || len(list) != 2 {
		t.Fatalf("дубли не схлопнулись: %#v", v)
	}
}

func TestCoerceAnswerTextValidation(t *testing.T) {
	q := Question{
		ID: 3, Type: QShortText, Title: "Возраст",
		Config: map[string]any{"validation": map[string]any{"kind": "number", "min": 18.0, "max": 100.0}},
	}

	if _, err := q.CoerceAnswer("сорок"); err == nil {
		t.Fatal("нечисло должно отвергаться числовой проверкой")
	}
	if _, err := q.CoerceAnswer("17"); err == nil {
		t.Fatal("значение ниже минимума должно отвергаться")
	}
	if v, err := q.CoerceAnswer(" 42 "); err != nil || v != "42" {
		t.Fatalf("законное число отвергнуто или не обрезано: %v, %v", v, err)
	}
}

func TestCoerceAnswerScaleBounds(t *testing.T) {
	q := Question{ID: 4, Type: QScale, Config: map[string]any{"min": 1, "max": 5}}
	if _, err := q.CoerceAnswer(9); err == nil {
		t.Fatal("значение вне шкалы должно отвергаться")
	}
	if v, err := q.CoerceAnswer(3); err != nil || v != 3 {
		t.Fatalf("законное значение шкалы: %v, %v", v, err)
	}
}

func TestCoerceAnswerGrid(t *testing.T) {
	q := Question{
		ID: 5, Type: QGridRadio,
		Config: map[string]any{
			"rows": []any{"Скорость", "Качество"},
			"cols": []any{"Хорошо", "Плохо"},
		},
	}

	if _, err := q.CoerceAnswer(map[string]any{"Скорость": "Идеально"}); err == nil {
		t.Fatal("столбец вне сетки должен отвергаться")
	}
	// Строку могли убрать из сетки после ответа — старое значение просто уходит.
	v, err := q.CoerceAnswer(map[string]any{"Скорость": "Хорошо", "Цена": "Плохо"})
	if err != nil {
		t.Fatalf("законная сетка отвергнута: %v", err)
	}
	got, _ := v.(map[string]any)
	if len(got) != 1 || got["Скорость"] != "Хорошо" {
		t.Fatalf("значение чужой строки не отброшено: %#v", got)
	}

	q.Config["require_each_row"] = true
	if _, err := q.CoerceAnswer(map[string]any{"Скорость": "Хорошо"}); err == nil {
		t.Fatal("незаполненная строка должна отвергаться при require_each_row")
	}
}

func TestGradeQuiz(t *testing.T) {
	radio := Question{
		ID: 1, Type: QRadio, Points: 5,
		Config:    map[string]any{"options": []any{"Да", "Нет"}},
		AnswerKey: map[string]any{"value": "Да"},
	}
	if got := Grade(radio, "Да"); got != 5 {
		t.Fatalf("верный ответ = %d, ожидалось 5", got)
	}
	if got := Grade(radio, "Нет"); got != 0 {
		t.Fatalf("неверный ответ = %d, ожидалось 0", got)
	}

	// Множественный выбор засчитывается только целиком: половина верных
	// вариантов — это неверный ответ.
	checks := Question{
		ID: 2, Type: QCheckbox, Points: 3,
		Config:    map[string]any{"options": []any{"A", "B", "C"}},
		AnswerKey: map[string]any{"values": []any{"A", "B"}},
	}
	if got := Grade(checks, []any{"A"}); got != 0 {
		t.Fatalf("неполный набор = %d, ожидалось 0", got)
	}
	if got := Grade(checks, []any{"B", "A"}); got != 3 {
		t.Fatalf("полный набор в другом порядке = %d, ожидалось 3", got)
	}

	// Текст сверяется без учёта регистра и пробелов по краям.
	text := Question{
		ID: 3, Type: QShortText, Points: 2,
		AnswerKey: map[string]any{"values": []any{"Париж"}},
	}
	if got := Grade(text, " париж "); got != 2 {
		t.Fatalf("текст с иным регистром = %d, ожидалось 2", got)
	}

	if total := MaxScore([]Question{radio, checks, text}); total != 10 {
		t.Fatalf("максимум = %d, ожидалось 10", total)
	}
}

func TestNormalizeStripsForeignSettings(t *testing.T) {
	// Ветвление держится на вариантах: у типа без выбора его быть не должно.
	q := Question{Type: QParagraph, Config: map[string]any{"targets": map[string]any{"a": "submit"}}}
	q.Normalize()
	if _, ok := q.Config["targets"]; ok {
		t.Fatal("переходы остались у типа без вариантов")
	}

	// Пояснительный блок ничего не спрашивает.
	note := Question{Type: QNote, Required: true, Points: 5,
		AnswerKey: map[string]any{"value": "x"}}
	note.Normalize()
	if note.Required || note.Points != 0 || len(note.AnswerKey) != 0 {
		t.Fatalf("пояснение осталось вопросом: %#v", note)
	}
}

func TestAnswerFilesAndCleanup(t *testing.T) {
	answers := map[string]any{
		"1": []any{
			map[string]any{"path": "forms/a.pdf", "name": "a.pdf", "thumb": "forms/a-thumb.jpg"},
			map[string]any{"path": "forms/b.pdf", "name": "b.pdf"},
		},
		"2": "просто текст",
	}
	if files := AnswerFiles(answers); len(files) != 2 {
		t.Fatalf("найдено файлов: %d, ожидалось 2", len(files))
	}

	next, changed, removed := AnswersWithoutFiles(answers, map[string]bool{"forms/a.pdf": true})
	if !changed {
		t.Fatal("удаление файла не отмечено")
	}
	// Миниатюра уходит вместе с оригиналом — иначе она осталась бы висеть в квоте.
	if len(removed) != 2 {
		t.Fatalf("удалено ключей: %v, ожидались файл и его миниатюра", removed)
	}
	if list, _ := next["1"].([]any); len(list) != 1 {
		t.Fatalf("во вложениях осталось %d файлов, ожидался один", len(list))
	}
}
