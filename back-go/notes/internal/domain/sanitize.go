package domain

import (
	"bytes"
	"encoding/json"
)

// protoKey — ключ, который TipTap v2 (mergeAttributes) превращает в
// УНАСЛЕДОВАННЫЕ атрибуты DOM: документ с attrs.__proto__.onerror исполнял бы
// обработчик у каждого, кто откроет заметку (GHSA для @tiptap/core ≤ 3.30).
// Своих атрибутов с таким именем у редактора нет, поэтому ключ просто вырезаем.
const protoKey = "__proto__"

// SanitizeDoc — документ без ключей __proto__ на любой глубине. Документ без
// подозрительных байтов возвращается как есть: полный разбор нужен, только
// если ключ мог встретиться (в том числе записанный escape-последовательностями
// — JSON.parse на клиенте их раскроет).
func SanitizeDoc(doc json.RawMessage) (json.RawMessage, error) {
	if !bytes.Contains(doc, []byte(protoKey)) && !bytes.Contains(doc, []byte(`\u`)) {
		return doc, nil
	}
	dec := json.NewDecoder(bytes.NewReader(doc))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	if !stripProto(v) {
		return doc, nil
	}
	return json.Marshal(v)
}

// stripProto — рекурсивно убирает protoKey; true — что-то было вырезано.
func stripProto(v any) bool {
	changed := false
	switch t := v.(type) {
	case map[string]any:
		if _, ok := t[protoKey]; ok {
			delete(t, protoKey)
			changed = true
		}
		for _, c := range t {
			if stripProto(c) {
				changed = true
			}
		}
	case []any:
		for _, c := range t {
			if stripProto(c) {
				changed = true
			}
		}
	}
	return changed
}
