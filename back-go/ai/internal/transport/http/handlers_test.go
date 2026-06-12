package http

import (
	"testing"
)

// Формы валидации PUT-тела — как marshmallow AiSettingsUpdateSchema.
func TestParseSettingsUpdate(t *testing.T) {
	t.Run("полный валидный боди", func(t *testing.T) {
		upd, details := parseSettingsUpdate([]byte(
			`{"enabled": true, "api_key": "sk-1", "clear_key": false,
			  "model_chat": "gpt-4o", "model_embedding": "text-embedding-3-small"}`))
		if details != nil {
			t.Fatalf("неожиданные ошибки: %v", details)
		}
		if upd.Enabled == nil || !*upd.Enabled || upd.APIKey == nil || *upd.APIKey != "sk-1" ||
			upd.ClearKey || upd.ModelChat == nil || *upd.ModelChat != "gpt-4o" {
			t.Fatalf("разбор: %+v", upd)
		}
	})

	t.Run("пустой и невалидный JSON — как {}", func(t *testing.T) {
		for _, body := range []string{"", "{broken"} {
			upd, details := parseSettingsUpdate([]byte(body))
			if details != nil || upd.Enabled != nil || upd.APIKey != nil {
				t.Fatalf("%q: ожидался пустой апдейт, %+v / %v", body, upd, details)
			}
		}
	})

	t.Run("api_key null — не менять", func(t *testing.T) {
		upd, details := parseSettingsUpdate([]byte(`{"api_key": null}`))
		if details != nil || upd.APIKey != nil {
			t.Fatalf("null api_key: %+v / %v", upd, details)
		}
	})

	t.Run("ошибки валидации в формате marshmallow", func(t *testing.T) {
		cases := []struct{ body, field, msg string }{
			{`{"enabled": "kinda"}`, "enabled", "Not a valid boolean."},
			{`{"api_key": 5}`, "api_key", "Not a valid string."},
			{`{"model_chat": ""}`, "model_chat", "Length must be between 1 and 64."},
			{`{"model_embedding": 7}`, "model_embedding", "Not a valid string."},
			{`{"unknown_field": 1}`, "unknown_field", "Unknown field."},
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
