package report

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestDocBuildsHeadingsAndTable(t *testing.T) {
	d := New()
	d.Heading("Активность сотрудника", 1)
	d.Para("Иванов Иван")
	d.Table([]string{"Показатель", "Значение"}, [][]string{
		{"Отработано часов", "12.5"},
		{"Закрыто задач", "3"},
	})
	data, err := d.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var doc string
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			rc.Close()
			doc = string(b)
		}
	}
	for _, want := range []string{"<w:tbl>", "Активность сотрудника", "Отработано часов", "12.5", "<w:b/>"} {
		if !strings.Contains(doc, want) {
			t.Errorf("document.xml не содержит %q", want)
		}
	}
	// Экранирование спецсимволов.
	d2 := New()
	d2.Para("a & b < c")
	data2, _ := d2.Bytes()
	if bytes.Contains(data2, []byte("a & b < c")) {
		t.Error("спецсимволы не экранированы")
	}
}
