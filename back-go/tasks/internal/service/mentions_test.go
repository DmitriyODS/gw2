package service

import (
	"reflect"
	"testing"
)

func TestParseMentionLogins(t *testing.T) {
	cases := []struct {
		text string
		want []string
	}{
		{"", nil},
		{"привет, @ivanov.i.i посмотри", []string{"ivanov.i.i"}},
		{"@a и @b и снова @a", []string{"a", "b"}},
		{"пиши на foo@bar.com", nil},          // e-mail — не упоминание
		{"конец предложения @petrov.", []string{"petrov"}}, // точка на конце обрезается
		{"@ПётрИванов кириллица", []string{"пётриванов"}},
		{"(в скобках)@sidorov", []string{"sidorov"}},
	}
	for _, c := range cases {
		got := parseMentionLogins(c.text)
		if len(got) == 0 && len(c.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseMentionLogins(%q) = %v, want %v", c.text, got, c.want)
		}
	}
}
