package records

import "testing"

func numberField(config map[string]any) FieldInfo {
	return FieldInfo{ID: 5, Type: FieldNumber, Label: "Количество", Config: config}
}

// Главная причина падений раздела: в числовом поле оказывались буквы, и
// сортировка по нему отвечала 500 на каждую страницу списка.
func TestValidateNumber_RejectsNonNumeric(t *testing.T) {
	f := numberField(nil)
	for _, bad := range []string{"Возможен заказ в типографии", "1 шт", "12,5", "abc", "1e5", "--3"} {
		if err := ValidateValue(f, bad); err == nil {
			t.Errorf("значение %q должно быть отвергнуто", bad)
		}
	}
	for _, good := range []string{"0", "12", "-3", "12.5", ".5", "+7", " 42 "} {
		if err := ValidateValue(f, good); err != nil {
			t.Errorf("значение %q должно приниматься, получено: %v", good, err)
		}
	}
	// Пустое значение — «не заполнено», это законно.
	if err := ValidateValue(f, ""); err != nil {
		t.Errorf("пустое значение должно приниматься: %v", err)
	}
}

// Всё, что прошло валидацию, обязано приводиться к numeric — иначе сторож в
// SQL сортировки увёл бы значение в NULL, и оно потерялось бы в конце списка.
func TestValidateNumber_AcceptedValuesAreCastable(t *testing.T) {
	f := numberField(nil)
	for _, s := range []string{"0", "12", "-3", "12.5", ".5", "+7"} {
		if err := ValidateValue(f, s); err != nil {
			t.Fatalf("%q не прошло валидацию: %v", s, err)
		}
		if _, ok := ParseNumber(s); !ok {
			t.Errorf("%q прошло валидацию, но числом не разбирается", s)
		}
	}
}

func TestValidateNumber_Bounds(t *testing.T) {
	f := numberField(map[string]any{"min": float64(0)})
	if err := ValidateValue(f, "-1"); err == nil {
		t.Error("значение ниже минимума должно быть отвергнуто")
	}
	if err := ValidateValue(f, "0"); err != nil {
		t.Errorf("ноль допустим при min=0: %v", err)
	}

	// Граница может приехать из JSONB строкой — читаем и её.
	f = numberField(map[string]any{"min": "0", "max": "10"})
	if err := ValidateValue(f, "11"); err == nil {
		t.Error("значение выше максимума должно быть отвергнуто")
	}
	if err := ValidateValue(f, "10"); err != nil {
		t.Errorf("верхняя граница включительна: %v", err)
	}

	// Пустая граница из формы — «предела нет».
	f = numberField(map[string]any{"min": "", "max": ""})
	if err := ValidateValue(f, "-100"); err != nil {
		t.Errorf("без границ принимается любое число: %v", err)
	}
}
