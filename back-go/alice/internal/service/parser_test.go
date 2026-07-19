package service

import (
	"testing"
	"time"
)

var now = time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC) // пятница

func TestParseTasks(t *testing.T) {
	cases := []struct {
		cmd   string
		kind  string
		title string
	}{
		{"добавь задачу подготовить отчет", "task_create", "подготовить отчет"},
		{"создай новую задачу позвонить клиенту", "task_create", "позвонить клиенту"},
		{"закрой задачу отчет за июнь", "task_close", "отчет за июнь"},
		{"мои задачи", "task_list", ""},
		{"начни работу над задаче отчет", "unit_start", "отчет"},
		{"начни работать над отчетом", "unit_start", "отчетом"},
		{"останови работу", "unit_stop", ""},
		{"что сейчас в работе", "unit_status", ""},
	}
	for _, c := range cases {
		it := Parse(c.cmd, now)
		if it.Kind != c.kind || it.Title != c.title {
			t.Errorf("%q → %q/%q, ожидалось %q/%q", c.cmd, it.Kind, it.Title, c.kind, c.title)
		}
	}
}

func TestParseDiary(t *testing.T) {
	it := Parse("запиши на завтра позвонить маме", now)
	if it.Kind != "diary_add" || it.Title != "позвонить маме" || it.Date != "2026-07-18" {
		t.Fatalf("diary_add: %+v", it)
	}
	it = Parse("добавь в ежедневник купить хлеб", now)
	if it.Kind != "diary_add" || it.Title != "купить хлеб" || it.Date != "" {
		t.Fatalf("diary_add без даты: %+v", it)
	}
	it = Parse("что у меня на сегодня", now)
	if it.Kind != "diary_list" || it.Date != "2026-07-17" {
		t.Fatalf("diary_list: %+v", it)
	}
	it = Parse("отметь купить хлеб выполненным", now)
	if it.Kind != "diary_done" || it.Title != "купить хлеб" {
		t.Fatalf("diary_done: %+v", it)
	}
	it = Parse("перенеси купить хлеб на понедельник", now)
	if it.Kind != "diary_move" || it.Title != "купить хлеб" || it.Date != "2026-07-20" {
		t.Fatalf("diary_move: %+v", it)
	}
	it = Parse("удали запись купить хлеб", now)
	if it.Kind != "diary_delete" || it.Title != "купить хлеб" {
		t.Fatalf("diary_delete: %+v", it)
	}
	it = Parse("создай ежедневник работа", now)
	if it.Kind != "diary_create" || it.Title != "работа" {
		t.Fatalf("diary_create: %+v", it)
	}
}

func TestParseNotes(t *testing.T) {
	it := Parse("создай заметку идеи с текстом сделать навык алисы", now)
	if it.Kind != "note_create" || it.Title != "идеи" || it.Text != "сделать навык алисы" {
		t.Fatalf("note_create: %+v", it)
	}
	it = Parse("допиши в заметку список покупок текст молоко и хлеб", now)
	if it.Kind != "note_append" || it.Title != "список покупок" || it.Text != "молоко и хлеб" {
		t.Fatalf("note_append: %+v", it)
	}
	it = Parse("прочитай заметку идеи", now)
	if it.Kind != "note_read" || it.Title != "идеи" {
		t.Fatalf("note_read: %+v", it)
	}
	it = Parse("удали заметку идеи", now)
	if it.Kind != "note_delete" || it.Title != "идеи" {
		t.Fatalf("note_delete: %+v", it)
	}
	it = Parse("создай папку работа", now)
	if it.Kind != "folder_create" || it.Title != "работа" {
		t.Fatalf("folder_create: %+v", it)
	}
}

func TestParseServiceIntents(t *testing.T) {
	if Parse("помощь", now).Kind != "help" {
		t.Fatal("help")
	}
	if Parse("да", now).Kind != "yes" {
		t.Fatal("yes")
	}
	if Parse("отмена", now).Kind != "no" {
		t.Fatal("no")
	}
	if Parse("расскажи анекдот", now).Kind != "unknown" {
		t.Fatal("unknown")
	}
}

func TestExtractDate(t *testing.T) {
	date, cleaned, ok := ExtractDate("купить хлеб на 15 января", now)
	if !ok || date != "2027-01-15" || cleaned != "купить хлеб" {
		t.Fatalf("прошедшая дата года: %s %q %v", date, cleaned, ok)
	}
	date, _, ok = ExtractDate("в пятницу созвон", now)
	if !ok || date != "2026-07-17" { // сегодня пятница → сегодня
		t.Fatalf("день недели: %s", date)
	}
}

func TestChoiceIndex(t *testing.T) {
	if ParseChoiceIndex("второй") != 2 || ParseChoiceIndex("вариант 3") != 3 || ParseChoiceIndex("нету") != 0 {
		t.Fatal("choice index")
	}
}
