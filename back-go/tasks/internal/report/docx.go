// Package report — минимальная генерация .docx (OOXML) без внешних зависимостей
// для отчётов активности: заголовки, абзацы и таблицы. Картинки не нужны.
package report

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strconv"
	"strings"
)

const contentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
	`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
	`<Default Extension="xml" ContentType="application/xml"/>` +
	`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
	`</Types>`

const rels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
	`</Relationships>`

// Doc — накопитель тела документа.
type Doc struct {
	body strings.Builder
}

func New() *Doc { return &Doc{} }

// Heading — заголовок уровня 1..3 (жирный, с интервалами).
func (d *Doc) Heading(text string, level int) {
	sz := map[int]int{1: 36, 2: 30, 3: 26}[level]
	if sz == 0 {
		sz = 26
	}
	d.body.WriteString(`<w:p><w:pPr><w:spacing w:before="240" w:after="120"/></w:pPr>` +
		`<w:r><w:rPr><w:b/><w:sz w:val="` + strconv.Itoa(sz) + `"/></w:rPr><w:t xml:space="preserve">` +
		esc(text) + `</w:t></w:r></w:p>`)
}

// Para — обычный абзац (пустая строка — вертикальный отступ).
func (d *Doc) Para(text string) {
	d.body.WriteString(`<w:p><w:r><w:t xml:space="preserve">` + esc(text) + `</w:t></w:r></w:p>`)
}

// Table — таблица с жирной шапкой и границами.
func (d *Doc) Table(headers []string, rows [][]string) {
	d.body.WriteString(`<w:tbl><w:tblPr><w:tblW w:w="0" w:type="auto"/><w:tblBorders>`)
	for _, side := range []string{"top", "left", "bottom", "right", "insideH", "insideV"} {
		d.body.WriteString(`<w:` + side + ` w:val="single" w:sz="4" w:space="0" w:color="auto"/>`)
	}
	d.body.WriteString(`</w:tblBorders></w:tblPr>`)
	d.row(headers, true)
	for _, r := range rows {
		d.row(r, false)
	}
	d.body.WriteString(`</w:tbl><w:p/>`)
}

func (d *Doc) row(cells []string, header bool) {
	d.body.WriteString(`<w:tr>`)
	for _, c := range cells {
		d.body.WriteString(`<w:tc><w:tcPr><w:tcW w:w="0" w:type="auto"/>`)
		if header {
			d.body.WriteString(`<w:shd w:val="clear" w:fill="F2F2F2"/>`)
		}
		d.body.WriteString(`</w:tcPr><w:p>`)
		if header {
			d.body.WriteString(`<w:r><w:rPr><w:b/></w:rPr><w:t xml:space="preserve">` + esc(c) + `</w:t></w:r>`)
		} else {
			d.body.WriteString(`<w:r><w:t xml:space="preserve">` + esc(c) + `</w:t></w:r>`)
		}
		d.body.WriteString(`</w:p></w:tc>`)
	}
	d.body.WriteString(`</w:tr>`)
}

// Bytes — собрать .docx.
func (d *Doc) Bytes() ([]byte, error) {
	body := d.body.String()
	if body == "" {
		body = `<w:p/>`
	}
	document := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
		body + `<w:sectPr/></w:body></w:document>`

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	parts := []struct{ name, data string }{
		{"[Content_Types].xml", contentTypes},
		{"_rels/.rels", rels},
		{"word/document.xml", document},
	}
	for _, p := range parts {
		w, err := zw.Create(p.name)
		if err != nil {
			return nil, err
		}
		if _, err := io.WriteString(w, p.data); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func esc(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}
