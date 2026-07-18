// Package docx — минимальная генерация и разбор .docx (OOXML) без внешних
// зависимостей: заметки состоят из абзацев плоского текста, чего достаточно для
// экспорта/импорта. Документ .docx — это zip с несколькими XML-частями.
package docx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strings"
)

const contentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`

const rels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`

// Build — .docx из заголовка (жирным) и плоского текста (по абзацу на строку).
func Build(title, text string) ([]byte, error) {
	var body strings.Builder
	if strings.TrimSpace(title) != "" {
		body.WriteString(`<w:p><w:r><w:rPr><w:b/><w:sz w:val="32"/></w:rPr><w:t xml:space="preserve">`)
		body.WriteString(escapeXML(title))
		body.WriteString(`</w:t></w:r></w:p>`)
	}
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		body.WriteString(`<w:p><w:r><w:t xml:space="preserve">`)
		body.WriteString(escapeXML(line))
		body.WriteString(`</w:t></w:r></w:p>`)
	}
	document := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
		body.String() + `<w:sectPr/></w:body></w:document>`

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

// Parse — плоский текст из .docx: текст всех <w:t>, новая строка на конце каждого
// абзаца <w:p>, <w:br/>/<w:tab/> — перенос/табуляция.
func Parse(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	var doc *zip.File
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			doc = f
			break
		}
	}
	if doc == nil {
		return "", nil
	}
	rc, err := doc.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()
	dec := xml.NewDecoder(rc)
	var b strings.Builder
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		switch t := tok.(type) {
		case xml.CharData:
			b.Write(t)
		case xml.StartElement:
			switch t.Name.Local {
			case "br":
				b.WriteByte('\n')
			case "tab":
				b.WriteByte('\t')
			}
		case xml.EndElement:
			if t.Name.Local == "p" {
				b.WriteByte('\n')
			}
		}
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

func escapeXML(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}
