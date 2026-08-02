package markdown

import (
	"encoding/json"
	"strings"
	"testing"
)

// docOf — распакованный документ для проверок структуры.
func docOf(t *testing.T, raw json.RawMessage) mdNode {
	t.Helper()
	var root mdNode
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatalf("разбор документа: %v", err)
	}
	return root
}

func TestToDocBlocks(t *testing.T) {
	doc := docOf(t, ToDoc(strings.Join([]string{
		"# Заголовок",
		"",
		"Обычный абзац",
		"со второй строкой",
		"",
		"## Подзаголовок",
		"",
		"- первый",
		"- второй",
		"",
		"1. раз",
		"2. два",
		"",
		"- [ ] не сделано",
		"- [x] сделано",
		"",
		"> цитата",
		"",
		"```go",
		"fmt.Println(1)",
		"```",
		"",
		"---",
		"",
		"| a | b |",
		"| --- | --- |",
		"| 1 | 2 |",
	}, "\n")))

	want := []string{
		"heading", "paragraph", "heading", "bulletList", "orderedList",
		"taskList", "blockquote", "codeBlock", "horizontalRule", "table",
	}
	if len(doc.Content) != len(want) {
		t.Fatalf("блоков %d, ждали %d: %+v", len(doc.Content), len(want), doc.Content)
	}
	for i, kind := range want {
		if doc.Content[i].Type != kind {
			t.Errorf("блок %d — %s, ждали %s", i, doc.Content[i].Type, kind)
		}
	}

	// Абзац из двух строк склеен мягким переносом.
	para := doc.Content[1]
	if len(para.Content) != 3 || para.Content[1].Type != "hardBreak" {
		t.Errorf("абзац без мягкого переноса: %+v", para.Content)
	}
	// Чек-лист помнит отметки.
	tasks := doc.Content[5]
	if checked, _ := tasks.Content[0].Attrs["checked"].(bool); checked {
		t.Error("первый пункт чек-листа не должен быть отмечен")
	}
	if checked, _ := tasks.Content[1].Attrs["checked"].(bool); !checked {
		t.Error("второй пункт чек-листа должен быть отмечен")
	}
	// Язык фенса сохраняется.
	if lang, _ := doc.Content[7].Attrs["language"].(string); lang != "go" {
		t.Errorf("язык кода %q, ждали go", lang)
	}
}

func TestToDocInlineMarks(t *testing.T) {
	doc := docOf(t, ToDoc("**жирный** и *курсив*, ~~зачёркнутый~~, `код`, ==выделение== и [ссылка](https://ex.com)"))
	marks := map[string]string{}
	var walk func(n mdNode)
	walk = func(n mdNode) {
		for _, m := range n.Marks {
			marks[m.Type] = n.Text
		}
		for _, c := range n.Content {
			walk(c)
		}
	}
	walk(doc)

	for kind, text := range map[string]string{
		"bold": "жирный", "italic": "курсив", "strike": "зачёркнутый",
		"code": "код", "highlight": "выделение", "link": "ссылка",
	} {
		if marks[kind] != text {
			t.Errorf("марка %s на тексте %q, ждали %q", kind, marks[kind], text)
		}
	}
}

// Вложенный список уходит подсписком в последний пункт — по отступу.
func TestToDocNestedList(t *testing.T) {
	doc := docOf(t, ToDoc("- верх\n  - вложенный\n- второй"))
	list := doc.Content[0]
	if list.Type != "bulletList" || len(list.Content) != 2 {
		t.Fatalf("список: %+v", list)
	}
	first := list.Content[0]
	if len(first.Content) != 2 || first.Content[1].Type != "bulletList" {
		t.Fatalf("вложенного списка нет: %+v", first.Content)
	}
}

// Круговой прогон: разметка переживает выгрузку и повторный импорт.
func TestRoundTrip(t *testing.T) {
	src := strings.Join([]string{
		"# Заголовок",
		"",
		"Текст с **жирным**, *курсивом* и [ссылкой](https://ex.com).",
		"",
		"- пункт",
		"- ещё пункт",
		"",
		"1. раз",
		"2. два",
		"",
		"- [x] сделано",
		"",
		"> цитата",
		"",
		"```go",
		"fmt.Println(1)",
		"```",
		"",
		"| a | b |",
		"| --- | --- |",
		"| 1 | 2 |",
	}, "\n")

	first := FromDoc(ToDoc(src))
	second := FromDoc(ToDoc(first))
	if first != second {
		t.Errorf("повторный прогон меняет документ:\n--- первый ---\n%s\n--- второй ---\n%s", first, second)
	}
	for _, want := range []string{
		"# Заголовок", "**жирным**", "*курсивом*", "[ссылкой](https://ex.com)",
		"- пункт", "1. раз", "- [x] сделано", "> цитата", "```go", "| a | b |",
	} {
		if !strings.Contains(first, want) {
			t.Errorf("в выгрузке нет %q:\n%s", want, first)
		}
	}
}

func TestFromDocEmpty(t *testing.T) {
	if got := FromDoc(nil); got != "" {
		t.Errorf("пустой документ дал %q", got)
	}
	if got := FromDoc(json.RawMessage(`{"broken`)); got != "" {
		t.Errorf("битый документ дал %q", got)
	}
}

// Картинка отдельной строкой — блочный узел (её ключ хранилища нужен экспорту).
func TestToDocLoneImage(t *testing.T) {
	doc := docOf(t, ToDoc("![схема](/uploads/notes/1.png)"))
	if doc.Content[0].Type != "image" {
		t.Fatalf("картинка не стала блоком: %+v", doc.Content[0])
	}
	if src, _ := doc.Content[0].Attrs["src"].(string); src != "/uploads/notes/1.png" {
		t.Errorf("src картинки %q", src)
	}
}
