// Package marshform — примитивы разбора JSON-тел в формах marshmallow:
// типы значений и тексты ошибок повторяют его дефолтные сообщения, чтобы
// REST-ответы Go-сервисов оставались байт-в-байт с прежними Flask-схемами.
package marshform

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Тексты ошибок marshmallow.
const (
	MsgRequired       = "Missing data for required field."
	MsgNotString      = "Not a valid string."
	MsgNotInteger     = "Not a valid integer."
	MsgNotBoolean     = "Not a valid boolean."
	MsgNotList        = "Not a valid list."
	MsgNotDate        = "Not a valid date."
	MsgNotDateTime    = "Not a valid datetime."
	MsgUnknownField   = "Unknown field."
)

// AsString — JSON-строка.
func AsString(raw json.RawMessage) (string, bool) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", false
	}
	return s, true
}

// AsBool — как marshmallow Boolean: bool, truthy/falsy-строки, числа 0/1.
func AsBool(raw json.RawMessage) (bool, bool) {
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		return b, true
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		switch strings.ToLower(s) {
		case "t", "true", "1", "on", "y", "yes":
			return true, true
		case "f", "false", "0", "off", "n", "no":
			return false, true
		}
		return false, false
	}
	var n float64
	if err := json.Unmarshal(raw, &n); err == nil {
		switch n {
		case 1:
			return true, true
		case 0:
			return false, true
		}
	}
	return false, false
}

// AsInt — как marshmallow Integer: число без дробной части или
// строка-число; bool — невалиден.
func AsInt(raw json.RawMessage) (int64, bool) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "true" || trimmed == "false" {
		return 0, false
	}
	var n float64
	if err := json.Unmarshal(raw, &n); err == nil {
		if n != float64(int64(n)) {
			return 0, false
		}
		return int64(n), true
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err != nil {
			return 0, false
		}
		return v, true
	}
	return 0, false
}

// IsNull — литерал null (для allow_none-полей).
func IsNull(raw json.RawMessage) bool {
	return strings.TrimSpace(string(raw)) == "null"
}

// LengthMax / LengthMin — сообщения validate.Length.
func LengthMax(max int) string {
	return "Longer than maximum length " + strconv.Itoa(max) + "."
}

func LengthMin(min int) string {
	return "Shorter than minimum length " + strconv.Itoa(min) + "."
}

// OneOf — сообщение validate.OneOf.
func OneOf(choices []string) string {
	return "Must be one of: " + strings.Join(choices, ", ") + "."
}
