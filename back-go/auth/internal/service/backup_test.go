package service

import "testing"

func TestParseArchiveV2(t *testing.T) {
	raw := []byte(`{"version":2,"sections":["auth"],"tables":{"users":[{"id":1}],"roles":[]}}`)
	a, err := parseArchive(raw)
	if err != nil {
		t.Fatalf("parseArchive: %v", err)
	}
	if a.Version != 2 {
		t.Fatalf("version = %d, want 2", a.Version)
	}
	if _, ok := a.Tables["users"]; !ok {
		t.Fatalf("table users missing")
	}
	if _, ok := a.Tables["roles"]; !ok {
		t.Fatalf("table roles missing")
	}
}

func TestParseArchiveLegacy(t *testing.T) {
	// Старый формат: таблицы на верхнем уровне массивами.
	raw := []byte(`{"roles":[{"id":1,"name":"Admin"}],"users":[{"id":1}]}`)
	a, err := parseArchive(raw)
	if err != nil {
		t.Fatalf("parseArchive: %v", err)
	}
	if len(a.Tables) != 2 {
		t.Fatalf("tables = %d, want 2", len(a.Tables))
	}
	if _, ok := a.Tables["roles"]; !ok {
		t.Fatalf("legacy table roles missing")
	}
}

func TestHasSection(t *testing.T) {
	if !hasSection([]string{"auth", "tasks"}, "auth") {
		t.Fatal("expected auth present")
	}
	if hasSection([]string{"tasks"}, "auth") {
		t.Fatal("expected auth absent")
	}
}
