package markdown

import (
	"encoding/json"
	"strconv"
	"strings"
)

// mdNode — проекция узла TipTap для выгрузки (та же форма, что у docx-билдера).
type mdNode struct {
	Type    string         `json:"type"`
	Text    string         `json:"text"`
	Attrs   map[string]any `json:"attrs"`
	Marks   []mdMark       `json:"marks"`
	Content []mdNode       `json:"content"`
}

type mdMark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs"`
}

// FromDoc — Markdown из документа TipTap. Подчёркивание записать в Markdown
// нечем — такой текст выгружается без выделения.
func FromDoc(doc json.RawMessage) string {
	var root mdNode
	if len(doc) == 0 || json.Unmarshal(doc, &root) != nil || root.Type != "doc" {
		return ""
	}
	var b strings.Builder
	writeBlocks(&b, root.Content, "")
	return strings.TrimSpace(b.String()) + "\n"
}

// writeBlocks — блоки одного уровня; indent приписывается каждой строке
// (вложенные списки и содержимое цитаты).
func writeBlocks(b *strings.Builder, nodes []mdNode, indent string) {
	for _, n := range nodes {
		writeBlock(b, n, indent)
	}
}

func writeBlock(b *strings.Builder, n mdNode, indent string) {
	switch n.Type {
	case "paragraph":
		if text := inlineText(n.Content); text != "" {
			writeLines(b, text, indent)
		}
		b.WriteString("\n")
	case "heading":
		lvl := attrInt(n.Attrs, "level", 1)
		if lvl < 1 || lvl > 6 {
			lvl = 1
		}
		writeLines(b, strings.Repeat("#", lvl)+" "+inlineText(n.Content), indent)
		b.WriteString("\n")
	case "bulletList", "orderedList", "taskList":
		writeList(b, n, indent)
		if indent == "" {
			b.WriteString("\n")
		}
	case "blockquote":
		var inner strings.Builder
		writeBlocks(&inner, n.Content, "")
		for _, line := range strings.Split(strings.TrimRight(inner.String(), "\n"), "\n") {
			b.WriteString(indent + strings.TrimRight("> "+line, " ") + "\n")
		}
		b.WriteString("\n")
	case "codeBlock":
		lang, _ := n.Attrs["language"].(string)
		b.WriteString(indent + "```" + lang + "\n")
		for _, line := range strings.Split(plainText(n.Content), "\n") {
			b.WriteString(indent + line + "\n")
		}
		b.WriteString(indent + "```\n\n")
	case "horizontalRule":
		b.WriteString(indent + "---\n\n")
	case "image":
		b.WriteString(indent + image(n) + "\n\n")
	case "table":
		writeTable(b, n, indent)
	}
}

// writeList — пункты списка; вложенные списки уходят рекурсией с отступом в
// два пробела (их понимает и наш разбор, и любой другой Markdown).
func writeList(b *strings.Builder, list mdNode, indent string) {
	num := attrInt(list.Attrs, "start", 1)
	for _, item := range list.Content {
		marker := "- "
		switch {
		case list.Type == "orderedList":
			marker = strconv.Itoa(num) + ". "
			num++
		case item.Type == "taskItem":
			marker = "- [ ] "
			if checked, _ := item.Attrs["checked"].(bool); checked {
				marker = "- [x] "
			}
		}
		first := true
		for _, child := range item.Content {
			switch child.Type {
			case "paragraph":
				text := inlineText(child.Content)
				if first {
					b.WriteString(indent + marker + text + "\n")
					first = false
				} else if text != "" {
					b.WriteString(indent + strings.Repeat(" ", len(marker)) + text + "\n")
				}
			case "bulletList", "orderedList", "taskList":
				writeList(b, child, indent+"  ")
			default:
				writeBlock(b, child, indent+"  ")
			}
		}
		if first { // пункт без абзацев — пустая строчка списка
			b.WriteString(indent + strings.TrimRight(marker, " ") + "\n")
		}
	}
}

func writeTable(b *strings.Builder, table mdNode, indent string) {
	rows := make([][]string, 0, len(table.Content))
	for _, r := range table.Content {
		if r.Type != "tableRow" {
			continue
		}
		cells := make([]string, 0, len(r.Content))
		for _, c := range r.Content {
			// Ячейка — свои абзацы; в строке таблицы переносов быть не может.
			cells = append(cells, strings.Join(strings.Fields(cellText(c)), " "))
		}
		rows = append(rows, cells)
	}
	if len(rows) == 0 {
		return
	}
	width := 0
	for _, r := range rows {
		if len(r) > width {
			width = len(r)
		}
	}
	line := func(cells []string) {
		b.WriteString(indent + "|")
		for i := 0; i < width; i++ {
			cell := ""
			if i < len(cells) {
				cell = cells[i]
			}
			b.WriteString(" " + cell + " |")
		}
		b.WriteString("\n")
	}
	line(rows[0])
	b.WriteString(indent + "|" + strings.Repeat(" --- |", width) + "\n")
	for _, r := range rows[1:] {
		line(r)
	}
	b.WriteString("\n")
}

func cellText(cell mdNode) string {
	parts := make([]string, 0, len(cell.Content))
	for _, c := range cell.Content {
		if c.Type == "paragraph" {
			parts = append(parts, inlineText(c.Content))
		} else {
			parts = append(parts, plainText(c.Content))
		}
	}
	return strings.Join(parts, " ")
}

// writeLines — многострочный текст (мягкие переносы) с общим отступом.
func writeLines(b *strings.Builder, text, indent string) {
	for _, line := range strings.Split(text, "\n") {
		b.WriteString(indent + line + "\n")
	}
}

// inlineText — строка Markdown из инлайн-содержимого абзаца.
func inlineText(nodes []mdNode) string {
	var b strings.Builder
	for _, n := range nodes {
		switch n.Type {
		case "text":
			b.WriteString(wrapMarks(n.Text, n.Marks))
		case "hardBreak":
			b.WriteString("\n")
		case "image":
			b.WriteString(image(n))
		default:
			b.WriteString(inlineText(n.Content))
		}
	}
	return strings.TrimRight(b.String(), " ")
}

// wrapMarks — обёртки марок вокруг текста. Код — самый внутренний (внутри него
// разметки нет), ссылка — самая внешняя.
func wrapMarks(text string, marks []mdMark) string {
	if text == "" {
		return ""
	}
	var href string
	code := false
	for _, m := range marks {
		if m.Type == "code" {
			code = true
		}
		if m.Type == "link" {
			href, _ = m.Attrs["href"].(string)
		}
	}
	if code {
		text = "`" + text + "`"
	}
	for _, m := range marks {
		switch m.Type {
		case "bold":
			text = "**" + text + "**"
		case "italic":
			text = "*" + text + "*"
		case "strike":
			text = "~~" + text + "~~"
		case "highlight":
			text = "==" + text + "=="
		}
	}
	if href != "" {
		text = "[" + text + "](" + href + ")"
	}
	return text
}

func image(n mdNode) string {
	src, _ := n.Attrs["src"].(string)
	alt, _ := n.Attrs["alt"].(string)
	return "![" + alt + "](" + src + ")"
}

// plainText — текст без разметки (тело кода).
func plainText(nodes []mdNode) string {
	var b strings.Builder
	for _, n := range nodes {
		if n.Text != "" {
			b.WriteString(n.Text)
		}
		if n.Type == "hardBreak" {
			b.WriteString("\n")
		}
		b.WriteString(plainText(n.Content))
	}
	return b.String()
}

func attrInt(attrs map[string]any, key string, dflt int) int {
	if attrs == nil {
		return dflt
	}
	switch v := attrs[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return dflt
}
