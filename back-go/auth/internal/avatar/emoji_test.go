package avatar

import (
	"strings"
	"testing"
)

func TestEmojiAvatarContainsGlyph(t *testing.T) {
	svg := string(EmojiAvatar(7, "🐙"))
	if !strings.Contains(svg, "🐙") {
		t.Fatalf("значок не попал в SVG: %s", svg)
	}
	if !strings.HasPrefix(svg, "<svg") {
		t.Fatalf("ожидался SVG: %s", svg[:40])
	}
	// Фон детерминирован от id: у разных людей одинаковый значок различим.
	if string(EmojiAvatar(8, "🐙")) == svg {
		t.Fatal("фон одинаков для разных пользователей")
	}
}

// Разметку в значке нужно экранировать: иначе строка из настроек попала бы в
// SVG как теги.
func TestEmojiAvatarEscapes(t *testing.T) {
	svg := string(EmojiAvatar(1, "<script>"))
	if strings.Contains(svg, "<script>") {
		t.Fatalf("разметка не экранирована: %s", svg)
	}
}

func TestNormalizeEmoji(t *testing.T) {
	cases := []struct {
		in   string
		out  string
		ok   bool
		note string
	}{
		{"🐙", "🐙", true, "обычный значок"},
		{"  🎯  ", "🎯", true, "пробелы по краям срезаются"},
		{"", "", true, "пусто — снять значок"},
		{"abc", "", false, "текст значком не считается"},
		{"🐙🐙🐙🐙🐙🐙🐙🐙🐙", "", false, "слишком длинная строка"},
	}
	for _, c := range cases {
		got, ok := NormalizeEmoji(c.in)
		if ok != c.ok || (ok && got != c.out) {
			t.Errorf("%s: NormalizeEmoji(%q) = (%q, %v), ждали (%q, %v)",
				c.note, c.in, got, ok, c.out, c.ok)
		}
	}
}
