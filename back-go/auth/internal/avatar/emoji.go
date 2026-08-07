package avatar

import (
	"crypto/sha256"
	"fmt"
	"html"
	"strconv"
	"strings"
	"unicode/utf8"
)

/* Аватар-эмодзи: человек выбирает значок вместо фотографии.

   Отдаём SVG, а не PNG: цветные эмодзи рисует шрифт системы, и своего
   эмодзи-шрифта серверу заводить не нужно (в Go его пришлось бы тащить
   мегабайтами и обновлять под новые символы). Адрес тот же, что у
   автоматического аватара, — поэтому эмодзи появляется везде, где показывается
   аватар, без правок в каждом месте интерфейса.

   Фон — детерминированный от id, как у identicon: одинаковый значок у разных
   людей всё равно различим. */

// MaxEmojiRunes — потолок длины: эмодзи бывают составными (флаги, семьи,
// модификаторы тона), но строка из десятка символов — уже не значок.
const MaxEmojiRunes = 8

// EmojiAvatar — SVG-аватар со значком на цветном круге.
func EmojiAvatar(userID int64, emoji string) []byte {
	data := sha256.Sum256([]byte(strconv.FormatInt(userID, 10)))
	hue := float64(data[0]) / 255.0
	bg := hslToRGB(hue, 0.62, 0.86)
	ring := hslToRGB(hue, 0.62, 0.62)

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 192 192" width="192" height="192">`+
		`<circle cx="96" cy="96" r="96" fill="#%02x%02x%02x"/>`+
		`<circle cx="96" cy="96" r="93" fill="none" stroke="#%02x%02x%02x" stroke-width="4"/>`+
		`<text x="96" y="96" text-anchor="middle" dominant-baseline="central" `+
		`font-size="104" font-family="'Apple Color Emoji','Segoe UI Emoji','Noto Color Emoji',sans-serif">%s</text>`+
		`</svg>`,
		bg.R, bg.G, bg.B, ring.R, ring.G, ring.B, html.EscapeString(emoji))
	return []byte(svg)
}

// NormalizeEmoji — обрезка и проверка значка. Пустая строка — снять эмодзи
// (вернётся обычный автоматический аватар).
func NormalizeEmoji(raw string) (string, bool) {
	emoji := strings.TrimSpace(raw)
	if emoji == "" {
		return "", true
	}
	// Буквы и цифры отсекаем: «аватар» из текста — это уже не значок, а способ
	// подделать чужую подпись.
	for _, r := range emoji {
		if r < 128 {
			return "", false
		}
	}
	return emoji, utf8.RuneCountInString(emoji) <= MaxEmojiRunes
}
