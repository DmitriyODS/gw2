package http

import (
	"testing"
)

// Разбор PUT-тела ИИ-настроек компании. Ключа у компании больше нет — есть
// тумблеры возможностей: они работают на платформенном ключе и тратят токены
// создателя компании.
func TestParseSettingsUpdate(t *testing.T) {
	t.Run("полный валидный боди", func(t *testing.T) {
		upd, details := parseSettingsUpdate([]byte(
			`{"enabled": true, "shared": true, "feat_search": false, "feat_tv_fact": true}`))
		if details != nil {
			t.Fatalf("неожиданные ошибки: %v", details)
		}
		if upd.Enabled == nil || !*upd.Enabled ||
			upd.Shared == nil || !*upd.Shared ||
			upd.FeatSearch == nil || *upd.FeatSearch ||
			upd.FeatTVFact == nil || !*upd.FeatTVFact {
			t.Fatalf("разбор: %+v", upd)
		}
	})

	t.Run("пустой и невалидный JSON — как {}", func(t *testing.T) {
		for _, body := range []string{"", "{broken"} {
			upd, details := parseSettingsUpdate([]byte(body))
			if details != nil || upd.Enabled != nil || upd.Shared != nil {
				t.Fatalf("%q: ожидался пустой апдейт, %+v / %v", body, upd, details)
			}
		}
	})

	t.Run("ошибки валидации в формате marshmallow", func(t *testing.T) {
		cases := []struct{ body, field, msg string }{
			{`{"enabled": "kinda"}`, "enabled", "Not a valid boolean."},
			{`{"shared": 5}`, "shared", "Not a valid boolean."},
			{`{"feat_search": "kinda"}`, "feat_search", "Not a valid boolean."},
			{`{"unknown_field": 1}`, "unknown_field", "Unknown field."},
			// Ключ компании больше не принимается — это неизвестное поле.
			{`{"api_key": "sk-1"}`, "api_key", "Unknown field."},
		}
		for _, tc := range cases {
			_, details := parseSettingsUpdate([]byte(tc.body))
			msgs, ok := details[tc.field]
			if !ok || len(msgs) != 1 || msgs[0] != tc.msg {
				t.Errorf("%s: ожидалось {%q: [%q]}, получено %v", tc.body, tc.field, tc.msg, details)
			}
		}
	})
}
