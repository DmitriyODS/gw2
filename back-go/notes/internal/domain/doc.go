package domain

import (
	"encoding/json"
	"strings"
)

// docNode — минимальная проекция узла документа TipTap: текст листьев,
// вложенное содержимое и атрибуты (пути картинок).
type docNode struct {
	Type    string         `json:"type"`
	Text    string         `json:"text"`
	Attrs   map[string]any `json:"attrs"`
	Content []docNode      `json:"content"`
}

// blockTypes — узлы, после которых в плоском тексте начинается новая строка.
var blockTypes = map[string]bool{
	"paragraph": true, "heading": true, "codeBlock": true,
	"listItem": true, "taskItem": true, "tableCell": true, "tableHeader": true,
}

// DocText — плоский текст rich-документа TipTap (для поиска и txt-экспорта):
// текст листьев как есть, блочные узлы — с новой строки, подряд идущие пустые
// строки схлопываются.
func DocText(doc json.RawMessage) string {
	var root docNode
	if len(doc) == 0 || json.Unmarshal(doc, &root) != nil {
		return ""
	}
	var b strings.Builder
	walkText(&b, root)
	return collapseBlank(b.String())
}

func walkText(b *strings.Builder, n docNode) {
	if n.Text != "" {
		b.WriteString(n.Text)
	}
	if n.Type == "hardBreak" {
		b.WriteString("\n")
	}
	for _, child := range n.Content {
		walkText(b, child)
	}
	if blockTypes[n.Type] {
		b.WriteString("\n")
	}
}

// collapseBlank — убрать хвостовые пробелы строк и схлопнуть пустые строки
// (блочная вложенность TipTap даёт по \n на каждый уровень).
func collapseBlank(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	prevBlank := false
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		blank := line == ""
		if blank && prevBlank {
			continue
		}
		out = append(out, line)
		prevBlank = blank
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// DocFileKeys — ключи хранилища всех картинок документа (attrs со строками
// вида "/uploads/notes/..."): по ним чистятся файлы при удалении заметки.
func DocFileKeys(doc json.RawMessage) []string {
	var root docNode
	if len(doc) == 0 || json.Unmarshal(doc, &root) != nil {
		return nil
	}
	var keys []string
	walkFileKeys(&keys, root)
	return keys
}

func walkFileKeys(keys *[]string, n docNode) {
	for _, v := range n.Attrs {
		if s, ok := v.(string); ok && strings.HasPrefix(s, "/uploads/notes/") {
			*keys = append(*keys, strings.TrimPrefix(s, "/uploads/"))
		}
	}
	for _, child := range n.Content {
		walkFileKeys(keys, child)
	}
}

// TextToDoc — документ TipTap из плоского текста (импорт .txt): каждая строка —
// параграф, пустые строки — пустые параграфы.
func TextToDoc(text string) json.RawMessage {
	type node map[string]any
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	content := make([]node, 0, len(lines))
	for _, line := range lines {
		p := node{"type": "paragraph"}
		if line != "" {
			p["content"] = []node{{"type": "text", "text": line}}
		}
		content = append(content, p)
	}
	raw, _ := json.Marshal(node{"type": "doc", "content": content})
	return raw
}
