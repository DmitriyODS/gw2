package avatar

import (
	"bytes"
	"testing"
)

func TestIdenticonPNG(t *testing.T) {
	png1 := Identicon(1)
	if !bytes.HasPrefix(png1, []byte("\x89PNG")) {
		t.Fatal("не PNG")
	}
	// Детерминированность и уникальность по id.
	if !bytes.Equal(png1, Identicon(1)) {
		t.Fatal("identicon не детерминирован")
	}
	if bytes.Equal(png1, Identicon(2)) {
		t.Fatal("identicon одинаков для разных id")
	}
}
