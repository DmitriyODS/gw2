package docx

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"strings"
	"testing"
)

// tinyPNG — 2×3 PNG для проверки встраивания и разбора размеров.
func tinyPNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 2, 3))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func unzip(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]byte{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		rc.Close()
		out[f.Name] = b
	}
	return out
}

func TestBuildRichEmbedsImageAndTable(t *testing.T) {
	img := tinyPNG(t)
	doc := json.RawMessage(`{"type":"doc","content":[
		{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Заголовок"}]},
		{"type":"paragraph","content":[
			{"type":"text","marks":[{"type":"bold"}],"text":"жирный "},
			{"type":"text","marks":[{"type":"link","attrs":{"href":"https://ex.com"}}],"text":"ссылка"}
		]},
		{"type":"image","attrs":{"src":"/uploads/notes/pic.png"}},
		{"type":"bulletList","content":[{"type":"listItem","content":[
			{"type":"paragraph","content":[{"type":"text","text":"пункт"}]}]}]},
		{"type":"table","content":[{"type":"tableRow","content":[
			{"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"H"}]}]},
			{"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"text","text":"C"}]}]}
		]}]}
	]}`)

	fetch := func(key string) ([]byte, error) {
		if key != "notes/pic.png" {
			t.Fatalf("unexpected key %q", key)
		}
		return img, nil
	}
	data, err := BuildRich("Моя заметка", doc, fetch)
	if err != nil {
		t.Fatal(err)
	}
	parts := unzip(t, data)

	if _, ok := parts["word/media/image1.png"]; !ok {
		t.Fatal("картинка не встроена в word/media/")
	}
	if !bytes.Equal(parts["word/media/image1.png"], img) {
		t.Fatal("байты картинки искажены")
	}
	docXML := string(parts["word/document.xml"])
	for _, want := range []string{"<w:tbl>", "<w:drawing>", "r:embed=", "<w:hyperlink", "<w:b/>", "Заголовок"} {
		if !strings.Contains(docXML, want) {
			t.Errorf("document.xml не содержит %q", want)
		}
	}
	if !strings.Contains(string(parts["[Content_Types].xml"]), `Extension="png"`) {
		t.Error("content-types без Default для png")
	}
	rels := string(parts["word/_rels/document.xml.rels"])
	if !strings.Contains(rels, "media/image1.png") || !strings.Contains(rels, `TargetMode="External"`) {
		t.Error("document.xml.rels без image/hyperlink relationship")
	}
	// EMU высоты 3px = 3*9525.
	if !strings.Contains(docXML, `cy="28575"`) {
		t.Error("не рассчитаны размеры картинки")
	}
}

func TestBuildRichEmpty(t *testing.T) {
	data, err := BuildRich("", json.RawMessage(`{"type":"doc","content":[]}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	parts := unzip(t, data)
	if !strings.Contains(string(parts["word/document.xml"]), "<w:body>") {
		t.Fatal("нет тела документа")
	}
}
