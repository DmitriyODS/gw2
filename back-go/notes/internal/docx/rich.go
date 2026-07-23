package docx

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"path"
	"strconv"
	"strings"
)

// ImageFetcher — читатель байтов картинки по ключу хранилища (без "/uploads/").
// nil или ошибка — картинка пропускается (документ всё равно собирается).
type ImageFetcher func(key string) ([]byte, error)

// rNode — проекция узла документа TipTap с марками (для форматирования ранов).
type rNode struct {
	Type    string         `json:"type"`
	Text    string         `json:"text"`
	Attrs   map[string]any `json:"attrs"`
	Marks   []rMark        `json:"marks"`
	Content []rNode        `json:"content"`
}

type rMark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs"`
}

const (
	emuPerPx     = 9525    // 96 DPI: 1px = 9525 EMU
	maxImgWidth  = 5486400 // ~6" — ширина текстового блока A4 при полях 2.5см
	nsMain       = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	nsRel        = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	nsWpDraw     = "http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"
	nsDrawMain   = "http://schemas.openxmlformats.org/drawingml/2006/main"
	nsPic        = "http://schemas.openxmlformats.org/drawingml/2006/picture"
	relTypeImage = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/image"
	relTypeLink  = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink"
)

type mediaFile struct {
	name string
	data []byte
}

// richBuilder накапливает тело document.xml, медиа-части и их relationship-ы.
type richBuilder struct {
	body   strings.Builder
	rels   strings.Builder
	media  []mediaFile
	exts   map[string]string // расширение → content-type для [Content_Types].xml
	fetch  ImageFetcher
	relSeq int
	imgSeq int
}

// BuildRich — .docx из заголовка и rich-документа TipTap: абзацы с
// форматированием, заголовки, списки, цитаты, код, таблицы и встроенные
// картинки (fetch читает их байты; при ошибке картинка пропускается).
func BuildRich(title string, doc json.RawMessage, fetch ImageFetcher) ([]byte, error) {
	b := &richBuilder{exts: map[string]string{}, fetch: fetch}

	if strings.TrimSpace(title) != "" {
		b.body.WriteString(`<w:p><w:pPr><w:spacing w:after="200"/></w:pPr><w:r><w:rPr><w:b/><w:sz w:val="40"/></w:rPr><w:t xml:space="preserve">`)
		b.body.WriteString(escapeXML(title))
		b.body.WriteString(`</w:t></w:r></w:p>`)
	}

	var root rNode
	if len(doc) > 0 && json.Unmarshal(doc, &root) == nil && root.Type == "doc" {
		b.renderBlocks(root.Content)
	}
	// Пустое тело: хотя бы один абзац (валидность OOXML).
	if b.body.Len() == 0 {
		b.body.WriteString(`<w:p/>`)
	}
	return b.zip()
}

// renderBlocks — блочные узлы верхнего уровня и внутри контейнеров.
func (b *richBuilder) renderBlocks(nodes []rNode) {
	for _, n := range nodes {
		b.renderBlock(n)
	}
}

func (b *richBuilder) renderBlock(n rNode) {
	switch n.Type {
	case "paragraph":
		b.para(n.Content, paraOpts{})
	case "heading":
		lvl := attrInt(n.Attrs, "level", 1)
		sz := map[int]int{1: 36, 2: 30, 3: 26}[lvl]
		if sz == 0 {
			sz = 26
		}
		b.para(n.Content, paraOpts{headingSz: sz, before: 240, after: 120})
	case "bulletList", "orderedList":
		b.list(n, 0)
	case "taskList":
		b.taskList(n, 0)
	case "blockquote":
		for _, c := range n.Content {
			if c.Type == "paragraph" {
				b.para(c.Content, paraOpts{indent: 360, italic: true})
			} else {
				b.renderBlock(c)
			}
		}
	case "codeBlock":
		b.codeBlock(n)
	case "horizontalRule":
		b.body.WriteString(`<w:p><w:pPr><w:pBdr><w:bottom w:val="single" w:sz="6" w:space="1" w:color="auto"/></w:pBdr></w:pPr></w:p>`)
	case "image":
		b.image(n)
	case "table":
		b.table(n)
	}
}

// paraOpts — параметры абзаца (отступ, стиль заголовка, курсив и пр.).
type paraOpts struct {
	indent    int    // twips слева
	headingSz int    // half-points; 0 — обычный текст
	bold      bool   // жирный (шапка таблицы)
	italic    bool   // цитата
	before    int    // отступ до, twips
	after     int    // отступ после, twips
	prefix    string // маркер списка перед текстом
}

func (b *richBuilder) para(content []rNode, o paraOpts) {
	b.body.WriteString(`<w:p>`)
	if pPr := paraProps(o); pPr != "" {
		b.body.WriteString(pPr)
	}
	base := runStyle{bold: o.headingSz > 0 || o.bold, sz: o.headingSz, italic: o.italic}
	if o.prefix != "" {
		b.writeRun(o.prefix, nil, base)
	}
	b.renderInline(content, base)
	b.body.WriteString(`</w:p>`)
}

func paraProps(o paraOpts) string {
	var p strings.Builder
	if o.indent > 0 {
		p.WriteString(`<w:ind w:left="` + strconv.Itoa(o.indent) + `"/>`)
	}
	if o.before > 0 || o.after > 0 {
		p.WriteString(`<w:spacing`)
		if o.before > 0 {
			p.WriteString(` w:before="` + strconv.Itoa(o.before) + `"`)
		}
		if o.after > 0 {
			p.WriteString(` w:after="` + strconv.Itoa(o.after) + `"`)
		}
		p.WriteString(`/>`)
	}
	if p.Len() == 0 {
		return ""
	}
	return `<w:pPr>` + p.String() + `</w:pPr>`
}

// list — маркированный/нумерованный список (с вложенностью по depth).
func (b *richBuilder) list(n rNode, depth int) {
	ordered := n.Type == "orderedList"
	idx := attrInt(n.Attrs, "start", 1)
	for _, item := range n.Content {
		if item.Type != "listItem" {
			continue
		}
		marker := "•  "
		if ordered {
			marker = strconv.Itoa(idx) + ".  "
			idx++
		}
		b.listItem(item, depth, marker)
	}
}

func (b *richBuilder) taskList(n rNode, depth int) {
	for _, item := range n.Content {
		if item.Type != "taskItem" {
			continue
		}
		marker := "☐  "
		if attrBool(item.Attrs, "checked") {
			marker = "☑  "
		}
		b.listItem(item, depth, marker)
	}
}

// listItem — первый абзац с маркером, остальные блоки — с тем же отступом.
func (b *richBuilder) listItem(item rNode, depth int, marker string) {
	indent := 360 + depth*360
	first := true
	for _, c := range item.Content {
		switch c.Type {
		case "paragraph":
			pre := ""
			if first {
				pre = marker
			}
			b.para(c.Content, paraOpts{indent: indent, prefix: pre})
			first = false
		case "bulletList", "orderedList":
			b.list(c, depth+1)
		case "taskList":
			b.taskList(c, depth+1)
		default:
			b.renderBlock(c)
		}
	}
}

func (b *richBuilder) codeBlock(n rNode) {
	text := ""
	for _, c := range n.Content {
		text += c.Text
	}
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		b.body.WriteString(`<w:p><w:pPr><w:shd w:val="clear" w:fill="F2F2F2"/></w:pPr>`)
		b.writeRun(line, nil, runStyle{mono: true})
		b.body.WriteString(`</w:p>`)
	}
}

// renderInline — текстовые раны и переносы внутри абзаца.
func (b *richBuilder) renderInline(content []rNode, base runStyle) {
	for _, c := range content {
		switch c.Type {
		case "text":
			b.writeRun(c.Text, c.Marks, base)
		case "hardBreak":
			b.body.WriteString(`<w:r><w:br/></w:r>`)
		case "image": // на случай инлайн-картинки внутри абзаца
			b.image(c)
		}
	}
}

// runStyle — принудительные свойства рана поверх собственных марок текста.
type runStyle struct {
	bold, italic, mono bool
	sz                 int
}

func (b *richBuilder) writeRun(text string, marks []rMark, base runStyle) {
	if text == "" {
		return
	}
	href := ""
	for _, m := range marks {
		if m.Type == "link" {
			if h, ok := m.Attrs["href"].(string); ok {
				href = h
			}
		}
	}
	rPr := buildRPr(marks, base, href != "")
	run := `<w:r>` + rPr + `<w:t xml:space="preserve">` + escapeXML(text) + `</w:t></w:r>`
	if href != "" {
		rid := b.addHyperlink(href)
		b.body.WriteString(`<w:hyperlink r:id="` + rid + `">` + run + `</w:hyperlink>`)
		return
	}
	b.body.WriteString(run)
}

func buildRPr(marks []rMark, base runStyle, isLink bool) string {
	bold, italic, underline, strike, code, highlight := base.bold, base.italic, false, false, base.mono, false
	for _, m := range marks {
		switch m.Type {
		case "bold":
			bold = true
		case "italic":
			italic = true
		case "underline":
			underline = true
		case "strike":
			strike = true
		case "code":
			code = true
		case "highlight":
			highlight = true
		}
	}
	var p strings.Builder
	if bold {
		p.WriteString(`<w:b/>`)
	}
	if italic {
		p.WriteString(`<w:i/>`)
	}
	if underline || isLink {
		p.WriteString(`<w:u w:val="single"/>`)
	}
	if strike {
		p.WriteString(`<w:strike/>`)
	}
	if code {
		p.WriteString(`<w:rFonts w:ascii="Consolas" w:hAnsi="Consolas"/>`)
	}
	if highlight {
		p.WriteString(`<w:highlight w:val="yellow"/>`)
	}
	if isLink {
		p.WriteString(`<w:color w:val="0563C1"/>`)
	}
	if base.sz > 0 {
		p.WriteString(`<w:sz w:val="` + strconv.Itoa(base.sz) + `"/>`)
	}
	if p.Len() == 0 {
		return ""
	}
	return `<w:rPr>` + p.String() + `</w:rPr>`
}

// image — встроить картинку из хранилища как inline-drawing.
func (b *richBuilder) image(n rNode) {
	src, _ := n.Attrs["src"].(string)
	if b.fetch == nil || !strings.HasPrefix(src, "/uploads/notes/") {
		return
	}
	// отсечь query (?...) на всякий случай
	key := strings.TrimPrefix(src, "/uploads/")
	if i := strings.IndexByte(key, '?'); i >= 0 {
		key = key[:i]
	}
	data, err := b.fetch(key)
	if err != nil || len(data) == 0 {
		return
	}
	cx, cy := imageEMU(data)
	rid := b.addImage(key, data)
	b.imgSeq++
	id := strconv.Itoa(b.imgSeq)
	b.body.WriteString(`<w:p><w:r><w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0">`)
	b.body.WriteString(`<wp:extent cx="` + strconv.Itoa(cx) + `" cy="` + strconv.Itoa(cy) + `"/>`)
	b.body.WriteString(`<wp:docPr id="` + id + `" name="Picture ` + id + `"/>`)
	b.body.WriteString(`<a:graphic xmlns:a="` + nsDrawMain + `"><a:graphicData uri="` + nsPic + `">`)
	b.body.WriteString(`<pic:pic xmlns:pic="` + nsPic + `"><pic:nvPicPr><pic:cNvPr id="` + id + `" name="Picture ` + id + `"/><pic:cNvPicPr/></pic:nvPicPr>`)
	b.body.WriteString(`<pic:blipFill><a:blip r:embed="` + rid + `"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill>`)
	b.body.WriteString(`<pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="` + strconv.Itoa(cx) + `" cy="` + strconv.Itoa(cy) + `"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr>`)
	b.body.WriteString(`</pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r></w:p>`)
}

// imageEMU — размеры картинки в EMU (с ограничением ширины блока текста).
func imageEMU(data []byte) (int, int) {
	w, h := 400, 300
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(data)); err == nil && cfg.Width > 0 && cfg.Height > 0 {
		w, h = cfg.Width, cfg.Height
	}
	cx, cy := w*emuPerPx, h*emuPerPx
	if cx > maxImgWidth {
		cy = int(int64(cy) * int64(maxImgWidth) / int64(cx))
		cx = maxImgWidth
	}
	return cx, cy
}

// table — таблица TipTap (строки/ячейки, шапка — жирным).
func (b *richBuilder) table(n rNode) {
	b.body.WriteString(`<w:tbl><w:tblPr><w:tblW w:w="0" w:type="auto"/><w:tblBorders>`)
	for _, side := range []string{"top", "left", "bottom", "right", "insideH", "insideV"} {
		b.body.WriteString(`<w:` + side + ` w:val="single" w:sz="4" w:space="0" w:color="auto"/>`)
	}
	b.body.WriteString(`</w:tblBorders></w:tblPr>`)
	for _, row := range n.Content {
		if row.Type != "tableRow" {
			continue
		}
		b.body.WriteString(`<w:tr>`)
		for _, cell := range row.Content {
			header := cell.Type == "tableHeader"
			b.body.WriteString(`<w:tc><w:tcPr><w:tcW w:w="0" w:type="auto"/>`)
			if header {
				b.body.WriteString(`<w:shd w:val="clear" w:fill="F2F2F2"/>`)
			}
			b.body.WriteString(`</w:tcPr>`)
			wrote := false
			for _, c := range cell.Content {
				if c.Type == "paragraph" {
					b.para(c.Content, paraOpts{bold: header})
					wrote = true
				} else {
					b.renderBlock(c)
					wrote = true
				}
			}
			if !wrote { // ячейка обязана содержать хотя бы один абзац
				b.body.WriteString(`<w:p/>`)
			}
			b.body.WriteString(`</w:tc>`)
		}
		b.body.WriteString(`</w:tr>`)
	}
	b.body.WriteString(`</w:tbl><w:p/>`) // абзац после таблицы — требование OOXML
}

func (b *richBuilder) nextRelID() string {
	b.relSeq++
	return "rId" + strconv.Itoa(b.relSeq)
}

func (b *richBuilder) addImage(key string, data []byte) string {
	ext := strings.ToLower(path.Ext(key))
	if ext == "" {
		ext = ".png"
	}
	b.exts[strings.TrimPrefix(ext, ".")] = imageContentType(ext)
	name := fmt.Sprintf("image%d%s", len(b.media)+1, ext)
	b.media = append(b.media, mediaFile{name: name, data: data})
	rid := b.nextRelID()
	b.rels.WriteString(`<Relationship Id="` + rid + `" Type="` + relTypeImage + `" Target="media/` + name + `"/>`)
	return rid
}

func (b *richBuilder) addHyperlink(url string) string {
	rid := b.nextRelID()
	b.rels.WriteString(`<Relationship Id="` + rid + `" Type="` + relTypeLink + `" Target="` + escapeAttr(url) + `" TargetMode="External"/>`)
	return rid
}

// zip — собрать финальный .docx со всеми частями.
func (b *richBuilder) zip() ([]byte, error) {
	document := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:document xmlns:w="` + nsMain + `" xmlns:r="` + nsRel + `" xmlns:wp="` + nsWpDraw +
		`" xmlns:a="` + nsDrawMain + `" xmlns:pic="` + nsPic + `"><w:body>` +
		b.body.String() + `<w:sectPr/></w:body></w:document>`

	var ctExtra strings.Builder
	for ext, ct := range b.exts {
		ctExtra.WriteString(`<Default Extension="` + ext + `" ContentType="` + ct + `"/>`)
	}
	contentTypesXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>` +
		ctExtra.String() +
		`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
		`</Types>`

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	write := func(name, data string) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, data)
		return err
	}
	if err := write("[Content_Types].xml", contentTypesXML); err != nil {
		return nil, err
	}
	if err := write("_rels/.rels", rels); err != nil {
		return nil, err
	}
	if err := write("word/document.xml", document); err != nil {
		return nil, err
	}
	if b.rels.Len() > 0 {
		docRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			b.rels.String() + `</Relationships>`
		if err := write("word/_rels/document.xml.rels", docRels); err != nil {
			return nil, err
		}
	}
	for _, m := range b.media {
		w, err := zw.Create("word/media/" + m.name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(m.data); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func imageContentType(ext string) string {
	switch strings.ToLower(ext) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".svg":
		return "image/svg+xml"
	default:
		return "application/octet-stream"
	}
}

func escapeAttr(s string) string {
	r := strings.NewReplacer(`&`, "&amp;", `"`, "&quot;", `<`, "&lt;", `>`, "&gt;")
	return r.Replace(s)
}

func attrInt(m map[string]any, key string, def int) int {
	if m == nil {
		return def
	}
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return def
}

func attrBool(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	b, _ := m[key].(bool)
	return b
}
