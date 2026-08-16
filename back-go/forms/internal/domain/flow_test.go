package domain

import "testing"

// sections — три раздела: первый спрашивает, второй и третий — ветки ответа.
func flowSections() []Section {
	return []Section{
		{
			ID: 10, Position: 0, NextAction: NextNext,
			Questions: []Question{{
				ID: 1, Type: QRadio, Title: "Вы наш клиент?",
				Config: map[string]any{
					"options": []any{"Да", "Нет"},
					"targets": map[string]any{"Да": "20", "Нет": "30"},
				},
			}},
		},
		{
			ID: 20, Position: 1, NextAction: NextSubmit,
			Questions: []Question{{ID: 2, Type: QShortText, Title: "Что понравилось?", Required: true}},
		},
		{
			ID: 30, Position: 2, NextAction: NextSubmit,
			Questions: []Question{{ID: 3, Type: QShortText, Title: "Почему нет?", Required: true}},
		},
	}
}

func TestVisitedSectionsBranching(t *testing.T) {
	sections := flowSections()

	yes := VisitedSections(sections, map[string]any{"1": "Да"})
	if len(yes) != 2 || yes[1].ID != 20 {
		t.Fatalf("ветка «Да» прошла разделы %v, ожидались 10 → 20", ids(yes))
	}

	no := VisitedSections(sections, map[string]any{"1": "Нет"})
	if len(no) != 2 || no[1].ID != 30 {
		t.Fatalf("ветка «Нет» прошла разделы %v, ожидались 10 → 30", ids(no))
	}

	// Без ответа ветвление не срабатывает — идём по порядку.
	plain := VisitedSections(sections, map[string]any{})
	if len(plain) != 2 || plain[1].ID != 20 {
		t.Fatalf("без ответа прошли %v, ожидались 10 → 20", ids(plain))
	}
}

func TestVisitedSectionsBreaksCycle(t *testing.T) {
	// Составитель волен закольцевать переходы, но заполнение обязано завершаться.
	first := int64(2)
	sections := []Section{
		{ID: 1, NextAction: NextSection, NextSectionID: &[]int64{2}[0]},
		{ID: 2, NextAction: NextSection, NextSectionID: &first},
	}
	got := VisitedSections(sections, map[string]any{})
	if len(got) != 2 {
		t.Fatalf("цикл не оборвался: %v", ids(got))
	}
}

func TestMissingRequiredOnlyOnRoute(t *testing.T) {
	sections := flowSections()

	// Ответ по ветке «Да»: обязательный вопрос ветки «Нет» спрашивать нельзя —
	// человек её не видел.
	answers := map[string]any{"1": "Да", "2": "Скорость"}
	if missing := MissingRequired(sections, answers); missing != "" {
		t.Fatalf("потребован вопрос вне маршрута: %q", missing)
	}

	// А свой обязательный вопрос ветка требует.
	if missing := MissingRequired(sections, map[string]any{"1": "Да"}); missing != "Что понравилось?" {
		t.Fatalf("не потребован обязательный вопрос маршрута: %q", missing)
	}
}

func TestNextTargetSubmitWins(t *testing.T) {
	sections := []Section{
		{
			ID: 1, NextAction: NextNext,
			Questions: []Question{{
				ID: 7, Type: QDropdown,
				Config: map[string]any{
					"options": []any{"Хватит"},
					"targets": map[string]any{"Хватит": NextSubmit},
				},
			}},
		},
		{ID: 2, NextAction: NextSubmit},
	}
	got := VisitedSections(sections, map[string]any{"7": "Хватит"})
	if len(got) != 1 {
		t.Fatalf("переход «отправить» не оборвал маршрут: %v", ids(got))
	}
}

func TestParseTargetIndex(t *testing.T) {
	if i, ok := ParseTargetIndex("#3"); !ok || i != 3 {
		t.Fatalf("позиция раздела разобрана как %d (%v)", i, ok)
	}
	if _, ok := ParseTargetIndex("12"); ok {
		t.Fatal("идентификатор раздела принят за позицию")
	}
	if _, ok := ParseTargetIndex(NextSubmit); ok {
		t.Fatal("«отправить» принято за позицию")
	}
}

func ids(sections []Section) []int64 {
	out := make([]int64, 0, len(sections))
	for _, s := range sections {
		out = append(out, s.ID)
	}
	return out
}

// ── Условное отображение ─────────────────────────────────────────

func visibilitySections() []Section {
	source := int64(1)
	return []Section{
		{
			ID: 10, NextAction: NextNext,
			Questions: []Question{
				{
					ID: 1, Type: QRadio, Title: "Есть автомобиль?",
					Config: map[string]any{"options": []any{"Да", "Нет"}},
				},
				{
					ID: 2, Type: QShortText, Title: "Госномер", Required: true,
					// Вопрос показывается только тем, кто ответил «Да».
					Config: map[string]any{
						"visible_question_id": 1,
						"visible_values":      []any{"Да"},
					},
				},
			},
		},
		{
			// Раздел целиком скрыт тем же условием.
			ID: 20, Position: 1, NextAction: NextSubmit,
			VisibleQuestionID: &source, VisibleValues: []string{"Да"},
			Questions: []Question{{ID: 3, Type: QShortText, Title: "Где паркуетесь?", Required: true}},
		},
	}
}

func TestQuestionVisibility(t *testing.T) {
	sections := visibilitySections()
	q := sections[0].Questions[1]

	if QuestionVisible(q, map[string]any{"1": "Нет"}) {
		t.Fatal("вопрос показан при неподходящем ответе")
	}
	if !QuestionVisible(q, map[string]any{"1": "Да"}) {
		t.Fatal("вопрос скрыт при подходящем ответе")
	}
	// Без ответа на источник условие не выполнено.
	if QuestionVisible(q, map[string]any{}) {
		t.Fatal("вопрос показан без ответа на источник")
	}
}

func TestSectionVisibilitySkipsInRoute(t *testing.T) {
	sections := visibilitySections()

	hidden := VisitedSections(sections, map[string]any{"1": "Нет"})
	if len(hidden) != 1 || hidden[0].ID != 10 {
		t.Fatalf("скрытый раздел остался в маршруте: %v", ids(hidden))
	}
	shown := VisitedSections(sections, map[string]any{"1": "Да"})
	if len(shown) != 2 {
		t.Fatalf("раздел не показан при выполненном условии: %v", ids(shown))
	}
}

func TestMissingRequiredIgnoresHidden(t *testing.T) {
	sections := visibilitySections()

	// «Нет» — оба скрытых обязательных вопроса не спрашиваются.
	if missing := MissingRequired(sections, map[string]any{"1": "Нет"}); missing != "" {
		t.Fatalf("потребован скрытый вопрос: %q", missing)
	}
	// «Да» — вопрос стал видимым и обязателен.
	if missing := MissingRequired(sections, map[string]any{"1": "Да"}); missing != "Госномер" {
		t.Fatalf("не потребован показанный обязательный вопрос: %q", missing)
	}
}

func TestAnsweredQuestionsDropsHidden(t *testing.T) {
	sections := visibilitySections()
	got := AnsweredQuestions(sections, map[string]any{"1": "Нет"})
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("в пройденных вопросах остались скрытые: %d", len(got))
	}
}
