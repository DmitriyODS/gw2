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
	NormalizeSpans(&col, &row, &cfg, 3)
	if col != 1 || row != 1 || cfg == nil {
		t.Errorf("нормализация: col=%d row=%d cfg=%v", col, row, cfg)
	}
	col = 7
	NormalizeSpans(&col, &row, &cfg, 3)
	if col != 3 {
		t.Errorf("col должен ограничиваться 3, получено %d", col)
	}
	// Реестры делят карточку на четверти — там потолок свой.
	col = 7
	NormalizeSpans(&col, &row, &cfg, 4)
	if col != 4 {
		t.Errorf("col должен ограничиваться 4, получено %d", col)
	}
}

func TestValidateValue_NewTypes(t *testing.T) {
	cases := []struct {
		name  string
		field FieldInfo
		value any
		ok    bool
	}{
		{"почта верная", FieldInfo{Label: "Почта", Type: FieldEmail}, "a.b@mail.ru", true},
		{"почта без домена", FieldInfo{Label: "Почта", Type: FieldEmail}, "a.b@mail", false},
		{"почта пустая", FieldInfo{Label: "Почта", Type: FieldEmail}, "", true},
		{"телефон с разделителями", FieldInfo{Label: "Тел", Type: FieldPhone}, "+7 (912) 345-67-89", true},
		{"телефон из букв", FieldInfo{Label: "Тел", Type: FieldPhone}, "позвонить позже", false},
		{"регулярка подходит", FieldInfo{Label: "Код", Type: FieldRegex,
			Config: map[string]any{"pattern": `^[A-Z]{2}-\d{3}$`}}, "AB-123", true},
		{"регулярка не подходит", FieldInfo{Label: "Код", Type: FieldRegex,
			Config: map[string]any{"pattern": `^[A-Z]{2}-\d{3}$`}}, "ab-123", false},
		// Сломанный шаблон — недосмотр составителя реестра; заполняющий запись
		// не должен из-за него упереться в отказ на любом значении.
		{"регулярка сломана", FieldInfo{Label: "Код", Type: FieldRegex,
			Config: map[string]any{"pattern": `^[A-Z`}}, "что угодно", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateValue(c.field, c.value)
			if (err == nil) != c.ok {
				t.Errorf("ValidateValue(%v) = %v, ждали ok=%v", c.value, err, c.ok)
			}
		})
	}
}

func TestCoerceData_NormalizesPhone(t *testing.T) {
	fields := []FieldInfo{{ID: 1, Type: FieldPhone, Label: "Тел"}}
	out, err := CoerceData(fields, map[string]any{"1": "+7 (912) 345-67-89"})
	if err != nil {
		t.Fatalf("CoerceData: %v", err)
	}
	if out["1"] != "+79123456789" {
		t.Errorf("телефон должен храниться без разделителей, получено %q", out["1"])
	}
}

func TestDateConfig(t *testing.T) {
	// Прежняя тройка year/month_day/time из архивных копий.
	legacy := DateConfig(map[string]any{"year": true, "month_day": true, "time": false})
	if !legacy.Day || !legacy.Month || legacy.Hours {
		t.Errorf("устаревший конфиг разобран неверно: %+v", legacy)
	}
	// Части включаются по одной.
	only := DateConfig(map[string]any{
		"year": false, "month": false, "day": false,
		"hours": true, "minutes": true, "seconds": true,
	})
	if only.Year || only.Day || !only.Seconds {
		t.Errorf("гранулярный конфиг разобран неверно: %+v", only)
	}
	// Поле без единой части нечего показать — трактуем как полную дату.
	empty := DateConfig(map[string]any{
		"year": false, "month": false, "day": false,
		"hours": false, "minutes": false, "seconds": false,
	})
	if !empty.Year || !empty.Day || !empty.Minutes {
		t.Errorf("пустой конфиг должен давать полную дату: %+v", empty)
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
