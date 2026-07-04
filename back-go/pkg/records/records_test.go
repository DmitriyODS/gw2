package records

import "testing"

func TestCoerceData_ValidatesAndFilters(t *testing.T) {
	fields := []FieldInfo{
		{ID: 1, Type: FieldNumber, Label: "Код", Config: map[string]any{"pattern": `^\d{3}$`}},
		{ID: 2, Type: FieldSelect, Label: "Статус", Config: map[string]any{"options": []any{"Новый", "Готов"}}},
		{ID: 3, Type: FieldText, Label: "Имя", Config: map[string]any{}},
	}

	out, err := CoerceData(fields, map[string]any{
		"1": "123", "2": "Готов", "3": "Иван", "999": "мусор",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(out) != 3 {
		t.Errorf("неизвестные ключи должны отбрасываться, получено %v", out)
	}

	if _, err := CoerceData(fields, map[string]any{"1": "12x"}); err == nil {
		t.Error("число вне маски должно отклоняться")
	}
	if _, err := CoerceData(fields, map[string]any{"2": "Левый"}); err == nil {
		t.Error("вариант вне options должен отклоняться")
	}
}

func TestSearchText_OnlySearchableTypes(t *testing.T) {
	fields := []FieldInfo{
		{ID: 1, Type: FieldText},
		{ID: 2, Type: FieldCheckbox},
		{ID: 3, Type: FieldSelect},
	}
	got := SearchText(fields, map[string]any{
		"1": "привет", "2": true, "3": []any{"a", "b"},
	})
	if got != "привет a b" {
		t.Errorf("SearchText = %q", got)
	}
}

func TestNormalizeSpans(t *testing.T) {
	col, row := 0, 0
	var cfg map[string]any
	NormalizeSpans(&col, &row, &cfg)
	if col != 1 || row != 1 || cfg == nil {
		t.Errorf("нормализация: col=%d row=%d cfg=%v", col, row, cfg)
	}
	col = 7
	NormalizeSpans(&col, &row, &cfg)
	if col != 3 {
		t.Errorf("col должен ограничиваться 3, получено %d", col)
	}
}

func TestNewShareCode_UniqueHex(t *testing.T) {
	a, err := NewShareCode()
	if err != nil || len(a) != 32 {
		t.Fatalf("code=%q err=%v", a, err)
	}
	b, _ := NewShareCode()
	if a == b {
		t.Error("коды должны быть уникальными")
	}
}
