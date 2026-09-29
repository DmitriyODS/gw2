package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSanitizeDoc(t *testing.T) {
	cases := []struct {
		name, in string
		dirty    bool
	}{
		{"clean", `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"hi"}]}]}`, false},
		{"plain key", `{"type":"doc","content":[{"type":"image","attrs":{"src":"x","__proto__":{"onerror":"alert(1)"}}}]}`, true},
		{"escaped key", `{"type":"doc","content":[{"type":"image","attrs":{"__proto__":{"onerror":"alert(1)"}}}]}`, true},
		{"text mentions key", `{"type":"doc","content":[{"type":"text","text":"__proto__"}]}`, false},
	}
	for _, c := range cases {
		out, err := SanitizeDoc(json.RawMessage(c.in))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		var v any
		if err := json.Unmarshal(out, &v); err != nil {
			t.Fatalf("%s: invalid json: %v", c.name, err)
		}
		if strings.Contains(string(out), "onerror") {
			t.Errorf("%s: payload survived: %s", c.name, out)
		}
		if !c.dirty && string(out) != c.in {
			t.Errorf("%s: clean doc changed: %s", c.name, out)
		}
	}
}
