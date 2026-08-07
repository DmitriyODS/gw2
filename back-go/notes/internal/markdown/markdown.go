// Package markdown — перевод Markdown ⇄ документ TipTap (`notes.doc`).
//
// Свой разбор, без внешних зависимостей: набор правил зеркалит фронтовый
// `front/src/utils/markdown.js` (заголовки h1–h3, фенсы кода, цитаты, списки и
// чек-листы, таблицы, линейка; инлайн — **жирный**, *курсив*, ~~зачёркнутый~~,
// `код`, ==выделение==, ссылки и картинки), поэтому один и тот же файл
// одинаково понимают заметки и портал.
//
// Узлы и марки — те же, что принимает редактор заметок (StarterKit + underline,
// link, highlight, taskList, table, image): подчёркивание в Markdown записать
// нечем, поэтому при выгрузке оно теряется — остальное ходит туда-обратно.
package markdown

import (
	"encoding/json"
	"regexp"
	"strings"
)

type node = map[string]any

// ── Markdown → TipTap ────────────────────────────────────────────

var (
	fenceRe   = regexp.MustCompile("^```(.*)$")
	headingRe = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	hrRe      = regexp.MustCompile(`^\s*(-{3,}|\*{3,}|_{3,})\s*$`)
	quoteRe   = regexp.MustCompile(`^>\s?`)
	listRe    = regexp.MustCompile(`^(\s*)(?:([-*+])|(\d+)[.)])\s+(.*)$`)
	taskRe    = regexp.MustCompile(`^\[([ xX])\]\s+(.*)$`)
	rowRe     = regexp.MustCompile(`^\|.*\|\s*$`)
	delimRe   = regexp.MustCompile(`^\|[\s\-:|]+\|\s*$`)
)

// ToDoc — документ TipTap из Markdown-текста.
func ToDoc(md string) json.RawMessage {
	blocks := parseBlocks(splitLines(md))
	if len(blocks) == 0 {
		blocks = []node{{"type": "paragraph"}}
	}
	raw, _ := json.Marshal(node{"type": "doc", "content": blocks})
	return raw
}

func splitLines(s string) []string {
	return strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
}

func parseBlocks(lines []string) []node {
	out := make([]node, 0, len(lines))
	var para []string

	flushPara := func() {
		if len(para) == 0 {
			return
		}
		out = append(out, paragraph(inlineLines(para)))
		para = nil
	}

	for i := 0; i < len(lines); {
		line := lines[i]

		if m := fenceRe.FindStringSubmatch(line); m != nil {
			flushPara()
			lang := strings.TrimSpace(m[1])
			var buf []string
			for i++; i < len(lines) && !strings.HasPrefix(lines[i], "```"); i++ {
				buf = append(buf, lines[i])
			}
			i++ // закрывающий фенс
			out = append(out, codeBlock(strings.Join(buf, "\n"), lang))
			continue
		}

		if m := headingRe.FindStringSubmatch(line); m != nil {
			flushPara()
			lvl := len(m[1])
			if lvl > 3 { // редактор знает только три уровня
				lvl = 3
			}
			out = append(out, node{
				"type": "heading", "attrs": node{"level": lvl},
				"content": parseInline(m[2]),
			})
			i++
			continue
		}

		if hrRe.MatchString(line) {
			flushPara()
			out = append(out, node{"type": "horizontalRule"})
			i++
			continue
		}

		if quoteRe.MatchString(line) {
			flushPara()
			var buf []string
			for ; i < len(lines) && quoteRe.MatchString(lines[i]); i++ {
				buf = append(buf, quoteRe.ReplaceAllString(lines[i], ""))
			}
			inner := parseBlocks(buf)
			if len(inner) == 0 {
				inner = []node{{"type": "paragraph"}}
			}
			out = append(out, node{"type": "blockquote", "content": inner})
			continue
		}

		if listRe.MatchString(line) {
			flushPara()
			var list node
			list, i = parseList(lines, i, indentOf(lines[i]))
			out = append(out, list)
			continue
		}

		if rowRe.MatchString(line) && i+1 < len(lines) && delimRe.MatchString(lines[i+1]) {
			flushPara()
			var table node
			table, i = parseTable(lines, i)
			out = append(out, table)
			continue
		}

		if strings.TrimSpace(line) == "" {
			flushPara()
			i++
			continue
		}

		// Картинка отдельной строкой — блочный узел, а не абзац с картинкой.
		if img := loneImage(line); img != nil {
			flushPara()
			out = append(out, img)
			i++
			continue
		}

		para = append(para, line)
		i++
	}
	flushPara()
	return out
}

func indentOf(line string) int {
	m := listRe.FindStringSubmatch(line)
	if m == nil {
		return 0
	}
	return len(strings.ReplaceAll(m[1], "\t", "  "))
}

// parseList — список от строки i и все его вложенные уровни. Возвращает узел и
// индекс первой строки за списком. Вложенность определяется отступом: строка с
// бо́льшим отступом уходит подсписком в последний пункт.
func parseList(lines []string, i, indent int) (node, int) {
	m := listRe.FindStringSubmatch(lines[i])
	ordered := m[3] != ""
	items := []node{}
	task := false

	for i < len(lines) {
		lm := listRe.FindStringSubmatch(lines[i])
		if lm == nil {
			break
		}
		cur := indentOf(lines[i])
		if cur > indent { // вложенный список — в последний пункт
			if len(items) == 0 {
				break
			}
			var sub node
			sub, i = parseList(lines, i, cur)
			last := items[len(items)-1]
			last["content"] = append(last["content"].([]node), sub)
			continue
		}
		if cur < indent || (lm[3] != "") != ordered {
			break
		}
		text := lm[4]
		if tm := taskRe.FindStringSubmatch(text); tm != nil {
			task = true
			items = append(items, node{
				"type":  "taskItem",
				"attrs": node{"checked": tm[1] != " "},
				"content": []node{
					paragraph(parseInline(tm[2])),
				},
			})
		} else {
			items = append(items, node{
				"type":    "listItem",
				"content": []node{paragraph(parseInline(text))},
			})
		}
		i++
	}

	kind := "bulletList"
	switch {
	case task:
		kind = "taskList"
	case ordered:
		kind = "orderedList"
	}
	// Чек-лист и обычный список в одном блоке невозможны: taskList принимает
	// только taskItem — «обычные» пункты такого списка тоже становятся задачами.
	if kind == "taskList" {
		for _, it := range items {
			if it["type"] == "listItem" {
				it["type"] = "taskItem"
				it["attrs"] = node{"checked": false}
			}
		}
	}
	return node{"type": kind, "content": items}, i
}

func parseTable(lines []string, i int) (node, int) {
	head := splitRow(lines[i])
	i += 2 // шапка + разделитель
	rows := [][]string{}
	for i < len(lines) && rowRe.MatchString(lines[i]) {
		rows = append(rows, splitRow(lines[i]))
		i++
	}

	cell := func(kind, text string) node {
		return node{"type": kind, "content": []node{paragraph(parseInline(text))}}
	}
	content := []node{}
	hdr := make([]node, 0, len(head))
	for _, c := range head {
		hdr = append(hdr, cell("tableHeader", c))
	}
	content = append(content, node{"type": "tableRow", "content": hdr})
	for _, r := range rows {
		cells := make([]node, 0, len(head))
		for ci := range head {
			text := ""
			if ci < len(r) {
				text = r[ci]
			}
			cells = append(cells, cell("tableCell", text))
		}
		content = append(content, node{"type": "tableRow", "content": cells})
	}
	return node{"type": "table", "content": content}, i
}

func splitRow(line string) []string {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "|")
	s = strings.TrimSuffix(s, "|")
	parts := strings.Split(s, "|")
	for k := range parts {
		parts[k] = strings.TrimSpace(parts[k])
	}
	return parts
}

// loneImage — картинка, занимающая всю строку: в документе это блочный узел,
// а не абзац с картинкой внутри.
func loneImage(line string) node {
	s := strings.TrimSpace(line)
	loc := imageRe.FindStringSubmatchIndex(s)
	if loc == nil || loc[0] != 0 || loc[1] != len(s) {
		return nil
	}
	return node{"type": "image", "attrs": node{
		"src": s[loc[4]:loc[5]], "alt": s[loc[2]:loc[3]],
	}}
}

func paragraph(content []node) node {
	p := node{"type": "paragraph"}
	if len(content) > 0 {
		p["content"] = content
	}
	return p
}

func codeBlock(text, lang string) node {
	b := node{"type": "codeBlock"}
	if lang != "" {
		b["attrs"] = node{"language": lang}
	}
	if text != "" {
		b["content"] = []node{{"type": "text", "text": text}}
	}
	return b
}

// inlineLines — строки одного абзаца: между ними мягкий перенос (как <br> у
// фронтового парсера).
func inlineLines(lines []string) []node {
	out := []node{}
	for i, l := range lines {
		if i > 0 {
			out = append(out, node{"type": "hardBreak"})
		}
		out = append(out, parseInline(l)...)
	}
	return out
}

// ── Инлайн ───────────────────────────────────────────────────────

var (
	imageRe  = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)\)`)
	linkRe   = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`)
	codeRe   = regexp.MustCompile("`([^`]+)`")
	boldRe   = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	bold2Re  = regexp.MustCompile(`__([^_]+)__`)
	italRe   = regexp.MustCompile(`\*([^*]+)\*`)
	ital2Re  = regexp.MustCompile(`_([^_]+)_`)
	strikeRe = regexp.MustCompile(`~~([^~]+)~~`)
	markRe   = regexp.MustCompile(`==([^=]+)==`)
	urlRe    = regexp.MustCompile(`(?:https?://|www\.)[^\s<>` + "`" + `]+`)
)

type rule struct {
	re   *regexp.Regexp
	kind string
}

// Порядок важен только при равном начале совпадения: картинка раньше ссылки
// (её синтаксис — ссылка с «!»), жирный раньше курсива (** внутри *).
var inlineRules = []rule{
	{imageRe, "image"},
	{linkRe, "link"},
	{codeRe, "code"},
	{boldRe, "bold"},
	{bold2Re, "bold"},
	{strikeRe, "strike"},
	{markRe, "highlight"},
	{italRe, "italic"},
	{ital2Re, "italic"},
	{urlRe, "autolink"},
}

func parseInline(s string) []node {
	return inlineWithMarks(s, nil)
}

// inlineWithMarks — разбор строки с уже накопленными марками: находим самое
// раннее совпадение правил, левый кусок отдаём текстом, содержимое — рекурсией
// с добавленной маркой.
func inlineWithMarks(s string, marks []node) []node {
	out := []node{}
	for s != "" {
		best := -1
		var bestLoc []int
		var bestRule rule
		for _, r := range inlineRules {
			loc := r.re.FindStringSubmatchIndex(s)
			if loc == nil {
				continue
			}
			if best == -1 || loc[0] < best {
				best, bestLoc, bestRule = loc[0], loc, r
			}
		}
		if best == -1 {
			out = append(out, textNode(s, marks)...)
			break
		}
		if best > 0 {
			out = append(out, textNode(s[:best], marks)...)
		}

		switch bestRule.kind {
		case "image":
			out = append(out, node{"type": "image", "attrs": node{
				"src": s[bestLoc[4]:bestLoc[5]], "alt": s[bestLoc[2]:bestLoc[3]],
			}})
		case "link":
			href := s[bestLoc[4]:bestLoc[5]]
			out = append(out, inlineWithMarks(s[bestLoc[2]:bestLoc[3]],
				withMark(marks, node{"type": "link", "attrs": node{"href": href}}))...)
		case "autolink":
			raw := s[bestLoc[0]:bestLoc[1]]
			// Хвостовая пунктуация к адресу не относится.
			trimmed := strings.TrimRight(raw, ".,;:!?)")
			href := trimmed
			if strings.HasPrefix(href, "www.") {
				href = "https://" + href
			}
			out = append(out, textNode(trimmed,
				withMark(marks, node{"type": "link", "attrs": node{"href": href}}))...)
			bestLoc[1] = bestLoc[0] + len(trimmed)
		case "code":
			// Внутри кода разметки нет — текст как есть.
			out = append(out, textNode(s[bestLoc[2]:bestLoc[3]],
				withMark(marks, node{"type": "code"}))...)
		default:
			out = append(out, inlineWithMarks(s[bestLoc[2]:bestLoc[3]],
				withMark(marks, node{"type": bestRule.kind}))...)
		}
		s = s[bestLoc[1]:]
	}
	return out
}

func withMark(marks []node, m node) []node {
	next := make([]node, 0, len(marks)+1)
	next = append(next, marks...)
	return append(next, m)
}

func textNode(text string, marks []node) []node {
	if text == "" {
		return nil
	}
	n := node{"type": "text", "text": text}
	if len(marks) > 0 {
		n["marks"] = marks
	}
	return []node{n}
}
