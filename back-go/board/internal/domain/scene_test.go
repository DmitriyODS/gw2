package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSceneTextCollectsLabels(t *testing.T) {
	scene := json.RawMessage(`{"version":1,"objects":[
		{"id":"a","type":"text","x":10,"y":20,"text":"План спринта"},
		{"id":"b","type":"path","points":[0,0,10,10]},
		{"id":"c","type":"sticky","x":0,"y":0,"w":100,"h":80,"text":"Созвон в 15:00"}
	]}`)
	got := SceneText(scene)
	if !strings.Contains(got, "План спринта") || !strings.Contains(got, "Созвон в 15:00") {
		t.Fatalf("надписи и стикеры должны попадать в текст поиска, получено %q", got)
	}
}

func TestParseSceneToleratesBrokenJSON(t *testing.T) {
	s := ParseScene(json.RawMessage(`{ это не json `))
	if len(s.Objects) != 0 || s.Background != "grid" || len(s.Layers) != 1 {
		t.Fatalf("битая сцена должна давать пустую, получено %+v", s)
	}
}

func TestParseSceneLiftsOldSceneToLayers(t *testing.T) {
	// Сцены первой версии слоёв не знали — объекты обязаны попасть в базовый.
	s := ParseScene(json.RawMessage(`{"objects":[{"id":"a","type":"rect"}]}`))
	if len(s.Layers) != 1 || s.Objects[0].Layer != s.Layers[0].ID {
		t.Fatalf("объект должен переехать в базовый слой, получено %+v", s)
	}
}

func TestSceneSVGSkipsHiddenLayers(t *testing.T) {
	scene := json.RawMessage(`{"layers":[
		{"id":"l1","name":"Виден","visible":true},
		{"id":"l2","name":"Скрыт","visible":false}],
		"objects":[
		{"id":"a","type":"text","x":10,"y":10,"text":"видимая надпись","layer":"l1"},
		{"id":"b","type":"text","x":10,"y":40,"text":"скрытая надпись","layer":"l2"}]}`)
	svg := string(SceneSVG(scene, nil))
	if !strings.Contains(svg, "видимая надпись") {
		t.Fatalf("видимый слой должен попасть в SVG:\n%s", svg)
	}
	if strings.Contains(svg, "скрытая надпись") {
		t.Fatalf("скрытый слой не должен попадать в SVG:\n%s", svg)
	}
}

func TestStorageKeyStripsUploadsPrefix(t *testing.T) {
	// Клиент хранит в сцене готовый URL — хранилищу нужен голый ключ.
	if got := StorageKey("/uploads/boards/2026/07/x.png"); got != "boards/2026/07/x.png" {
		t.Fatalf("ожидался ключ без префикса, получено %q", got)
	}
	if got := StorageKey("boards/x.png"); got != "boards/x.png" {
		t.Fatalf("ключ без префикса не должен меняться, получено %q", got)
	}
}

func TestSceneTextIncludesCommentReplies(t *testing.T) {
	scene := json.RawMessage(`{"objects":[{"id":"c","type":"comment","x":0,"y":0,
		"text":"вопрос по макету","replies":[{"text":"ответ по макету"}]}]}`)
	got := SceneText(scene)
	if !strings.Contains(got, "вопрос по макету") || !strings.Contains(got, "ответ по макету") {
		t.Fatalf("обсуждение должно попадать в текст поиска, получено %q", got)
	}
}

func TestSceneImageKeys(t *testing.T) {
	scene := json.RawMessage(`{"objects":[
		{"id":"a","type":"image","src":"/uploads/boards/2026/07/x.png"},
		{"id":"b","type":"rect"}
	]}`)
	keys := SceneImageKeys(scene)
	if len(keys) != 1 || keys[0] != "boards/2026/07/x.png" {
		t.Fatalf("ожидался один ключ картинки, получено %v", keys)
	}
}

func TestSceneSVGRendersObjects(t *testing.T) {
	scene := json.RawMessage(`{"objects":[
		{"id":"a","type":"rect","x":0,"y":0,"w":100,"h":50,"color":"blue","fill":"amber"},
		{"id":"b","type":"arrow","x":0,"y":0,"x2":80,"y2":80,"color":"red"},
		{"id":"c","type":"text","x":10,"y":40,"text":"Привет & <мир>"}
	]}`)
	svg := string(SceneSVG(scene, nil))
	for _, want := range []string{"<svg", "<rect", "<line", "marker-end", "Привет &amp; &lt;мир&gt;"} {
		if !strings.Contains(svg, want) {
			t.Fatalf("в SVG нет %q:\n%s", want, svg)
		}
	}
}

func TestTextToSceneMakesLabels(t *testing.T) {
	raw := TextToScene("первая\n\nвторая")
	s := ParseScene(raw)
	if len(s.Objects) != 2 || s.Objects[0].Type != ObjText || s.Objects[1].Text != "вторая" {
		t.Fatalf("ожидались две надписи, получено %+v", s.Objects)
	}
}
