package domain

import (
	"encoding/json"
	"testing"
)

func TestDocText(t *testing.T) {
	doc := json.RawMessage(`{"type":"doc","content":[
		{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Заголовок"}]},
		{"type":"paragraph","content":[
			{"type":"text","text":"жирный","marks":[{"type":"bold"}]},
			{"type":"text","text":" и обычный"}]},
		{"type":"paragraph"},
		{"type":"paragraph","content":[
			{"type":"text","text":"строка"},{"type":"hardBreak"},{"type":"text","text":"перенос"}]},
		{"type":"bulletList","content":[
			{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"пункт"}]}]}]}
	]}`)
	got := DocText(doc)
	want := "Заголовок\nжирный и обычный\n\nстрока\nперенос\nпункт"
	if got != want {
		t.Fatalf("DocText:\n got %q\nwant %q", got, want)
	}
}

func TestDocTextEmptyAndInvalid(t *testing.T) {
	if got := DocText(nil); got != "" {
		t.Fatalf("nil doc: %q", got)
	}
	if got := DocText(json.RawMessage(`{}`)); got != "" {
		t.Fatalf("пустой doc: %q", got)
	}
	if got := DocText(json.RawMessage(`not-json`)); got != "" {
		t.Fatalf("битый doc: %q", got)
	}
}

func TestDocFileKeys(t *testing.T) {
	doc := json.RawMessage(`{"type":"doc","content":[
		{"type":"image","attrs":{"src":"/uploads/notes/a.png","alt":"x"}},
		{"type":"paragraph","content":[{"type":"image","attrs":{"src":"/uploads/notes/b.jpg"}}]},
		{"type":"image","attrs":{"src":"https://example.com/ext.png"}}
	]}`)
	keys := DocFileKeys(doc)
	if len(keys) != 2 || keys[0] != "notes/a.png" || keys[1] != "notes/b.jpg" {
		t.Fatalf("DocFileKeys: %v", keys)
	}
}

func TestTextToDocRoundTrip(t *testing.T) {
	doc := TextToDoc("первая\n\nтретья")
	if got := DocText(doc); got != "первая\n\nтретья" {
		t.Fatalf("round-trip: %q", got)
	}
}
