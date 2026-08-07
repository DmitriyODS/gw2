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

/* DocWithoutFiles — документ без узлов, ссылающихся на перечисленные ключи
   (человек убирает картинку из раздела «Настройки → Хранилище»). Второе
   значение — менялось ли что-нибудь.

   Работаем по сырому дереву map[string]any, а не по docNode: та — проекция
   только нужных полей, и обратная сборка из неё потеряла бы всё остальное
   (marks, выравнивания, атрибуты таблиц). */
func DocWithoutFiles(doc json.RawMessage, keys []string) (json.RawMessage, bool) {
	if len(doc) == 0 || len(keys) == 0 {
		return doc, false
	}
	var root any
	if json.Unmarshal(doc, &root) != nil {
		return doc, false
	}
	drop := make(map[string]bool, len(keys))
	for _, k := range keys {
		drop[k] = true
	}
	cleaned, changed := pruneFileNodes(root, drop)
	if !changed {
		return doc, false
	}
	out, err := json.Marshal(cleaned)
	if err != nil {
		return doc, false
	}
	return out, true
}

// pruneFileNodes — вырезать из content узлы с картинкой из drop.
func pruneFileNodes(node any, drop map[string]bool) (any, bool) {
	obj, ok := node.(map[string]any)
	if !ok {
		return node, false
	}
	content, ok := obj["content"].([]any)
	if !ok {
		return obj, false
	}
	changed := false
	kept := make([]any, 0, len(content))
	for _, child := range content {
		if nodeUsesFile(child, drop) {
			changed = true
			continue
		}
		cleaned, childChanged := pruneFileNodes(child, drop)
		changed = changed || childChanged
		kept = append(kept, cleaned)
	}
	if !changed {
		return obj, false
	}
	obj["content"] = kept
	return obj, true
}

func nodeUsesFile(node any, drop map[string]bool) bool {
	obj, ok := node.(map[string]any)
	if !ok {
		return false
	}
	attrs, ok := obj["attrs"].(map[string]any)
	if !ok {
		return false
	}
	for _, v := range attrs {
		if s, ok := v.(string); ok && drop[strings.TrimPrefix(s, "/uploads/")] {
			return true
		}
	}
	return false
}

// TextToDoc — документ TipTap из плоского текста (импорт .txt): каждая строка —
// параграф, пустые строки — пустые параграфы.
// AppendTextToDoc — дописать плоский текст абзацами в конец TipTap-документа
// (голосовое «допиши в заметку», навык Алисы). Пустой/невалидный документ
// заменяется новым из текста.
func AppendTextToDoc(doc json.RawMessage, text string) json.RawMessage {
	type docRoot struct {
		Type    string            `json:"type"`
		Content []json.RawMessage `json:"content"`
	}
	var root docRoot
	if len(doc) == 0 || json.Unmarshal(doc, &root) != nil || root.Type != "doc" {
		return TextToDoc(text)
	}
	var extra docRoot
	if json.Unmarshal(TextToDoc(text), &extra) != nil {
		return doc
	}
	root.Content = append(root.Content, extra.Content...)
	raw, err := json.Marshal(root)
	if err != nil {
		return doc
	}
	return raw
}

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
