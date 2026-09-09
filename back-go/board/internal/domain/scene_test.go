package domain

import (
	"encoding/json"
	"fmt"
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

func TestSceneSVGKeepsCustomHexColor(t *testing.T) {
	// Произвольный цвет уезжает в файл как есть, а мусор в атрибут не попадает.
	scene := json.RawMessage(`{"objects":[
		{"id":"a","type":"rect","x":0,"y":0,"w":10,"h":10,"color":"#A1B2C3"},
		{"id":"b","type":"rect","x":20,"y":0,"w":10,"h":10,"color":"red\" onload=\"alert(1)"}]}`)
	svg := string(SceneSVG(scene, nil))
	if !strings.Contains(svg, `stroke="#a1b2c3"`) {
		t.Fatalf("произвольный цвет должен попасть в SVG:\n%s", svg)
	}
	if strings.Contains(svg, "onload") {
		t.Fatalf("непроверенная строка не должна попадать в атрибут:\n%s", svg)
	}
}

func TestSceneSVGErasingStrokeBecomesMask(t *testing.T) {
	// Пиксельный ластик — не объект холста, а чёрный штрих в маске слоя.
	scene := json.RawMessage(`{"layers":[{"id":"l1","name":"Слой","visible":true}],"objects":[
		{"id":"a","type":"rect","x":0,"y":0,"w":100,"h":100,"color":"blue","layer":"l1"},
		{"id":"e","type":"path","points":[10,10,90,90],"width":20,"erase":true,"layer":"l1"}]}`)
	svg := string(SceneSVG(scene, nil))
	if !strings.Contains(svg, "<mask id=\"lm1\"") || !strings.Contains(svg, `stroke="#000000"`) {
		t.Fatalf("стирающий штрих должен уехать в маску:\n%s", svg)
	}
	if !strings.Contains(svg, `mask="url(#lm1)"`) {
		t.Fatalf("слой должен ссылаться на свою маску:\n%s", svg)
	}
}

func TestSceneSVGLayerOpacityBlendAndClip(t *testing.T) {
	scene := json.RawMessage(`{"layers":[
		{"id":"l1","name":"База","visible":true},
		{"id":"l2","name":"Верх","visible":true,"opacity":0.5,"blend":"multiply","clip":true}],
		"objects":[
		{"id":"a","type":"ellipse","x":0,"y":0,"w":50,"h":50,"color":"blue","layer":"l1"},
		{"id":"b","type":"rect","x":0,"y":0,"w":50,"h":50,"color":"red","layer":"l2"}]}`)
	svg := string(SceneSVG(scene, nil))
	if !strings.Contains(svg, `opacity="0.50"`) || !strings.Contains(svg, "mix-blend-mode:multiply") {
		t.Fatalf("прозрачность и наложение слоя должны попасть в SVG:\n%s", svg)
	}
	if !strings.Contains(svg, "mask-type:alpha") || !strings.Contains(svg, `mask="url(#am1)"`) {
		t.Fatalf("обтравка должна стать маской по альфе:\n%s", svg)
	}
}

func TestSceneSVGRotatesObject(t *testing.T) {
	scene := json.RawMessage(`{"objects":[{"id":"a","type":"rect","x":0,"y":0,"w":100,"h":40,"angle":30}]}`)
	svg := string(SceneSVG(scene, nil))
	if !strings.Contains(svg, `transform="rotate(30.0 50.0 20.0)"`) {
		t.Fatalf("поворот считается вокруг центра рамки:\n%s", svg)
	}
}

func TestSceneSVGExportsFirstFrameOnly(t *testing.T) {
	// В файл уходит первый кадр плюс объекты без кадра (фон анимации).
	scene := json.RawMessage(`{"animation":{"fps":12,"frames":[{"id":"f1"},{"id":"f2"}]},"objects":[
		{"id":"a","type":"text","x":0,"y":0,"text":"первый","frame":"f1"},
		{"id":"b","type":"text","x":0,"y":30,"text":"второй","frame":"f2"},
		{"id":"c","type":"text","x":0,"y":60,"text":"общий"}]}`)
	svg := string(SceneSVG(scene, nil))
	if !strings.Contains(svg, "первый") || !strings.Contains(svg, "общий") {
		t.Fatalf("первый кадр и общие объекты обязаны быть в SVG:\n%s", svg)
	}
	if strings.Contains(svg, "второй") {
		t.Fatalf("чужой кадр в файл попадать не должен:\n%s", svg)
	}
}

func TestParseSceneDropsUnknownFrameKeepsObject(t *testing.T) {
	// Кадр удалили — объект становится общим, а не исчезает вместе с ним.
	s := ParseScene(json.RawMessage(`{"animation":{"fps":12,"frames":[{"id":"f1"}]},
		"objects":[{"id":"a","type":"rect","frame":"нет-такого"}]}`))
	if len(s.Objects) != 1 || s.Objects[0].Frame != "" {
		t.Fatalf("объект чужого кадра должен стать общим, получено %+v", s.Objects)
	}
}

func TestParseSceneDropsClipOnBottomLayer(t *testing.T) {
	// Нижнему слою обтравляться не по чему — иначе он исчез бы целиком.
	s := ParseScene(json.RawMessage(`{"layers":[{"id":"l1","visible":true,"clip":true}],"objects":[]}`))
	if s.Layers[0].Clip {
		t.Fatalf("обтравка нижнего слоя должна сниматься, получено %+v", s.Layers[0])
	}
}

func TestSceneSVGSmoothPathUsesCurves(t *testing.T) {
	// Кисть рисуется квадратичными сегментами — как на холсте.
	scene := json.RawMessage(`{"objects":[{"id":"a","type":"path","points":[0,0,10,10,20,0,30,10],"smooth":true,"width":4}]}`)
	svg := string(SceneSVG(scene, nil))
	if !strings.Contains(svg, "Q") {
		t.Fatalf("сглаженный контур должен идти кривыми:\n%s", svg)
	}
}

func TestSceneSVGGradientGoesToDefs(t *testing.T) {
	scene := json.RawMessage(`{"objects":[{"id":"a","type":"rect","x":0,"y":0,"w":100,"h":40,
		"gradient":{"type":"linear","angle":90,"stops":[{"color":"blue","at":0},{"color":"#ff0000","at":1}]}}]}`)
	svg := string(SceneSVG(scene, nil))
	if !strings.Contains(svg, "<linearGradient") || !strings.Contains(svg, `stop-color="#3b74d6"`) {
		t.Fatalf("градиент должен уехать в defs:\n%s", svg)
	}
	if !strings.Contains(svg, `stop-color="#ff0000"`) {
		t.Fatalf("произвольный цвет точки градиента теряется:\n%s", svg)
	}
	if !strings.Contains(svg, `fill="url(#gr1)"`) {
		t.Fatalf("фигура должна заливаться градиентом:\n%s", svg)
	}
}

func TestSceneSVGEffectsBecomeFilter(t *testing.T) {
	scene := json.RawMessage(`{"objects":[{"id":"a","type":"rect","x":0,"y":0,"w":50,"h":50,
		"effects":{"blur":4,"saturate":2,"grayscale":0.5,"shadow":{"x":2,"y":4,"blur":8,"color":"ink","opacity":0.4}}}]}`)
	svg := string(SceneSVG(scene, nil))
	for _, want := range []string{"<filter", "feGaussianBlur", "feColorMatrix", "feDropShadow", `filter="url(#fx1)"`} {
		if !strings.Contains(svg, want) {
			t.Fatalf("в SVG нет %q:\n%s", want, svg)
		}
	}
}

func TestSceneSVGBooleanUnionAndExclude(t *testing.T) {
	// Однородная группа выражается ОДНИМ контуром: объединение — nonzero,
	// исключение — evenodd.
	base := `{"objects":[
		{"id":"a","type":"rect","x":0,"y":0,"w":50,"h":50,"color":"blue","bool":"g1","boolOp":"union"},
		{"id":"b","type":"ellipse","x":25,"y":25,"w":50,"h":50,"color":"red","bool":"g1","boolOp":"%s"}]}`
	union := string(SceneSVG(json.RawMessage(fmt.Sprintf(base, "union")), nil))
	if !strings.Contains(union, `fill-rule="nonzero"`) {
		t.Fatalf("объединение должно идти одним контуром:\n%s", union)
	}
	exclude := string(SceneSVG(json.RawMessage(fmt.Sprintf(base, "exclude")), nil))
	if !strings.Contains(exclude, `fill-rule="evenodd"`) {
		t.Fatalf("исключение должно идти по evenodd:\n%s", exclude)
	}
}

func TestSceneSVGBooleanSubtractUsesMask(t *testing.T) {
	scene := json.RawMessage(`{"objects":[
		{"id":"a","type":"rect","x":0,"y":0,"w":50,"h":50,"color":"blue","bool":"g1","boolOp":"union"},
		{"id":"b","type":"ellipse","x":25,"y":25,"w":50,"h":50,"bool":"g1","boolOp":"subtract"}]}`)
	svg := string(SceneSVG(scene, nil))
	if !strings.Contains(svg, `mask="url(#bm1)"`) || !strings.Contains(svg, "<mask id=\"bm1\"") {
		t.Fatalf("вычитание должно стать маской:\n%s", svg)
	}
	// Вычитаемая фигура сама не рисуется — только вырезает.
	if strings.Count(svg, "<ellipse") > 0 {
		t.Fatalf("вычитаемая фигура не должна рисоваться отдельно:\n%s", svg)
	}
}

func TestSceneSVGMaskObjectClipsWhatIsAbove(t *testing.T) {
	scene := json.RawMessage(`{"objects":[
		{"id":"m","type":"ellipse","x":0,"y":0,"w":80,"h":80,"fill":"ink","maskObject":true},
		{"id":"a","type":"rect","x":0,"y":0,"w":80,"h":80,"color":"red"}]}`)
	svg := string(SceneSVG(scene, nil))
	if !strings.Contains(svg, "mask-type:alpha") || !strings.Contains(svg, `mask="url(#am1)"`) {
		t.Fatalf("объект-маска должна обрезать то, что над ней:\n%s", svg)
	}
}

func TestSceneSVGVectorAndTextOnPath(t *testing.T) {
	scene := json.RawMessage(`{"objects":[
		{"id":"p","type":"vector","closed":false,"nodes":[
			{"x":0,"y":0,"ix":0,"iy":0,"ox":20,"oy":-20},
			{"x":100,"y":0,"ix":80,"iy":20,"ox":100,"oy":0}]},
		{"id":"t","type":"text","x":0,"y":0,"text":"по контуру","pathRef":"p","pathOffset":10}]}`)
	svg := string(SceneSVG(scene, nil))
	if !strings.Contains(svg, "C20.0 -20.0 80.0 20.0 100.0 0.0") {
		t.Fatalf("векторный контур должен идти кривыми Безье:\n%s", svg)
	}
	if !strings.Contains(svg, "<textPath") || !strings.Contains(svg, `startOffset="10.0%"`) {
		t.Fatalf("текст должен идти по контуру:\n%s", svg)
	}
}

func TestSceneSVGSkipsHiddenObjects(t *testing.T) {
	scene := json.RawMessage(`{"objects":[
		{"id":"a","type":"text","x":0,"y":0,"text":"видно"},
		{"id":"b","type":"text","x":0,"y":30,"text":"скрыто","hidden":true}]}`)
	svg := string(SceneSVG(scene, nil))
	if !strings.Contains(svg, "видно") || strings.Contains(svg, "скрыто") {
		t.Fatalf("скрытый объект не должен попадать в файл:\n%s", svg)
	}
}

func TestParseSceneDropsLonelyBoolGroup(t *testing.T) {
	// Булева группа из одного участника ничего не значит — метка снимается.
	s := ParseScene(json.RawMessage(`{"objects":[{"id":"a","type":"rect","bool":"g1","boolOp":"subtract"}]}`))
	if s.Objects[0].Bool != "" || s.Objects[0].BoolOp != "" {
		t.Fatalf("одинокая булева метка должна сниматься, получено %+v", s.Objects[0])
	}
}
