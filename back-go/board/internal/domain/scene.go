package domain

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
)

func base64Encode(data []byte) string { return base64.StdEncoding.EncodeToString(data) }

// Scene — содержимое доски: плоский список объектов холста. Клиент рисует их
// сам (Canvas/SVG), сервер знает ровно столько, сколько нужно ему самому:
// извлечь текст надписей для поиска и построить SVG при экспорте.
//
// Цвет объекта — КЛЮЧ палитры (ink/red/blue…), а не hex: фронт разворачивает
// ключ в токен --tag-*, поэтому доска остаётся в теме приложения и в тёмном
// режиме. Исключение — произвольный цвет («#rrggbb»), выбранный пипеткой или
// палитрой: он хранится значением и теме не следует. Конкретные значения нужны
// только серверу для SVG-экспорта (файл уезжает наружу) — sceneHex ниже.
type Scene struct {
	Version    int             `json:"version"`
	Background string          `json:"background"`
	Layers     []SceneLayer    `json:"layers"`
	Objects    []SceneObject   `json:"objects"`
	Animation  *SceneAnimation `json:"animation,omitempty"`
}

// SceneAnimation — покадровая анимация: кадры и частота. Кадр — метка у
// объектов (SceneObject.Frame), отдельного набора объектов у него нет.
type SceneAnimation struct {
	FPS    int          `json:"fps"`
	Frames []SceneFrame `json:"frames"`
}

type SceneFrame struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// SceneLayer — слой холста: порядок в срезе задаёт порядок отрисовки (первый —
// нижний), Visible скрывает слой целиком, Locked запрещает правку в клиенте.
// Opacity/Blend — прозрачность и режим наложения всего слоя, Clip — обтравка
// по слою снизу, Mask — многоугольная маска видимости.
type SceneLayer struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Visible bool          `json:"visible"`
	Locked  bool          `json:"locked"`
	Opacity *float64      `json:"opacity,omitempty"`
	Blend   string        `json:"blend,omitempty"`
	Clip    bool          `json:"clip,omitempty"`
	Mask    *SceneMask    `json:"mask,omitempty"`
	Effects *SceneEffects `json:"effects,omitempty"`
}

// SceneNode — узел векторного контура: точка и две направляющие Безье в
// абсолютных координатах сцены (зеркало utils/boardVector.js).
type SceneNode struct {
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
	IX float64 `json:"ix"`
	IY float64 `json:"iy"`
	OX float64 `json:"ox"`
	OY float64 `json:"oy"`
}

// SceneGradient — градиентная заливка: точки перехода в долях 0..1.
type SceneGradient struct {
	Type  string              `json:"type"`
	Angle float64             `json:"angle"`
	Stops []SceneGradientStop `json:"stops"`
}

type SceneGradientStop struct {
	Color string  `json:"color"`
	At    float64 `json:"at"`
}

// SceneEffects — коррекции и тень. Значения — доли, как в CSS-фильтрах
// (1 — «как есть»), чтобы холст и SVG считали одно и то же.
type SceneEffects struct {
	Blur       float64      `json:"blur,omitempty"`
	Brightness *float64     `json:"brightness,omitempty"`
	Contrast   *float64     `json:"contrast,omitempty"`
	Saturate   *float64     `json:"saturate,omitempty"`
	Hue        float64      `json:"hue,omitempty"`
	Grayscale  float64      `json:"grayscale,omitempty"`
	Sepia      float64      `json:"sepia,omitempty"`
	Shadow     *SceneShadow `json:"shadow,omitempty"`
}

type SceneShadow struct {
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Blur    float64 `json:"blur"`
	Color   string  `json:"color"`
	Opacity float64 `json:"opacity"`
}

// SceneMask — маска слоя: многоугольник в координатах сцены. Invert — вырезать
// область вместо того, чтобы оставить только её.
type SceneMask struct {
	Points []float64 `json:"points"`
	Invert bool      `json:"invert,omitempty"`
}

// SceneObject — объект холста. Набор полей общий для всех типов: лишние поля
// конкретного типа просто не заполняются (три одинаковых структуры-наследника
// дали бы больше кода, чем экономии).
type SceneObject struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	// X/Y/W/H — рамка объекта; для line/arrow — X,Y и X2,Y2 концы.
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
	W  float64 `json:"w,omitempty"`
	H  float64 `json:"h,omitempty"`
	X2 float64 `json:"x2,omitempty"`
	Y2 float64 `json:"y2,omitempty"`
	// Points — точки свободного пера/ломаной, парами [x0,y0,x1,y1,…].
	Points []float64 `json:"points,omitempty"`
	// Color/Fill — ключи палитры или «#rrggbb» (Fill "" — без заливки).
	Color   string  `json:"color,omitempty"`
	Fill    string  `json:"fill,omitempty"`
	Width   float64 `json:"width,omitempty"`
	Opacity float64 `json:"opacity,omitempty"`
	Text    string  `json:"text,omitempty"`
	Size    float64 `json:"size,omitempty"`
	// Angle — поворот вокруг центра рамки, в градусах.
	Angle float64 `json:"angle,omitempty"`
	// Nodes — узлы векторного контура (тип vector).
	Nodes []SceneNode `json:"nodes,omitempty"`
	// Gradient/Effects — градиентная заливка и коррекции с тенью.
	Gradient *SceneGradient `json:"gradient,omitempty"`
	Effects  *SceneEffects  `json:"effects,omitempty"`
	// Radius — скругление углов прямоугольных объектов.
	Radius *float64 `json:"radius,omitempty"`
	// Hidden/Locked — состояние строки в дереве слоёв (скрыт, заперт).
	Hidden bool `json:"hidden,omitempty"`
	Locked bool `json:"locked,omitempty"`
	// MaskObject — объект-маска: показывает то, что лежит над ним в слое,
	// только внутри своей формы (как «использовать как маску» в Figma).
	MaskObject bool `json:"maskObject,omitempty"`
	// Bool/BoolOp — булева группа и операция участника; первый участник в
	// порядке отрисовки — база, из неё вычитают и с ней пересекают.
	Bool   string `json:"bool,omitempty"`
	BoolOp string `json:"boolOp,omitempty"`
	// Текст по контуру: ссылка на объект-контур и сдвиг начала в процентах.
	PathRef    string  `json:"pathRef,omitempty"`
	PathOffset float64 `json:"pathOffset,omitempty"`
	// Erase — стирающий штрих: выедает нарисованное ниже него в СВОЁМ слое
	// (пиксельный ластик и «стереть внутри лассо»).
	Erase bool `json:"erase,omitempty"`
	// Smooth/Closed/Filled — сглаженная кривая, замкнутый контур, заливка
	// контура; Solid — заливка в полную силу, а не полупрозрачная.
	Smooth bool `json:"smooth,omitempty"`
	Closed bool `json:"closed,omitempty"`
	Filled bool `json:"filled,omitempty"`
	Solid  bool `json:"solid,omitempty"`
	// Многоугольник: число сторон, «звезда» и доля внутреннего радиуса.
	Sides int     `json:"sides,omitempty"`
	Star  bool    `json:"star,omitempty"`
	Inner float64 `json:"inner,omitempty"`
	// Src — адрес картинки, каким его положил клиент (/uploads/<key>).
	Src string `json:"src,omitempty"`
	// Layer — слой объекта, Group — id группы (объекты группы двигаются вместе),
	// Frame — кадр анимации (пусто — объект виден во всех кадрах).
	Layer string `json:"layer,omitempty"`
	Group string `json:"group,omitempty"`
	Frame string `json:"frame,omitempty"`
	// Комментарии: автор, ответы и пометка «решено» живут в самой сцене.
	Author   string       `json:"author,omitempty"`
	AuthorID int64        `json:"author_id,omitempty"`
	Resolved bool         `json:"resolved,omitempty"`
	Replies  []SceneReply `json:"replies,omitempty"`
	Created  string       `json:"created_at,omitempty"`
}

// SceneReply — ответ в обсуждении у булавки комментария.
type SceneReply struct {
	Author   string `json:"author,omitempty"`
	AuthorID int64  `json:"author_id,omitempty"`
	Text     string `json:"text"`
	Created  string `json:"created_at,omitempty"`
}

// Типы объектов холста (зеркало front/src/utils/boardScene.js).
const (
	ObjPath    = "path"    // свободное перо/кисть/кривая
	ObjVector  = "vector"  // векторный контур с узлами Безье
	ObjLine    = "line"    // прямая
	ObjArrow   = "arrow"   // стрелка
	ObjRect    = "rect"    // прямоугольник
	ObjEllipse = "ellipse" // эллипс
	ObjDiamond = "diamond" // ромб
	ObjPolygon = "polygon" // много-/звезда
	ObjText    = "text"    // надпись
	ObjSticky  = "sticky"  // липкая заметка
	ObjImage   = "image"   // картинка
	ObjComment = "comment" // булавка обсуждения
)

// commentPin — диаметр булавки комментария (зеркало COMMENT_PIN на фронте).
const commentPin = 28

// baseLayer — слой сцен первой версии, у которых слоёв ещё не было.
const baseLayer = "base"

func defaultLayers() []SceneLayer {
	return []SceneLayer{{ID: baseLayer, Name: "Слой 1", Visible: true, Locked: false}}
}

// SceneColors — палитра доски: ключ → цвет для SVG-экспорта. В приложении те же
// ключи разворачиваются в токены --tag-*, поэтому набор синхронен с палитрой
// тегов задач (front/src/utils/taskColors.js) плюс нейтральные «чернила» и
// белый (им рисуют по тёмному фону, поэтому он задан значением и в приложении).
var SceneColors = map[string]string{
	"ink":    "#1f2430",
	"white":  "#ffffff",
	"chalk":  "#f8fafc",
	"red":    "#e05252",
	"orange": "#e07a3c",
	"amber":  "#d9a520",
	"green":  "#3fa45b",
	"teal":   "#2b9b9b",
	"blue":   "#3b74d6",
	"violet": "#7a5cd6",
	"pink":   "#d65c9b",
}

// Фоны холста (сетка/точки/чистый лист) — зеркало фронта.
var SceneBackgrounds = map[string]bool{"grid": true, "dots": true, "plain": true}

// SceneBlends — режимы наложения слоя: значения совпадают с CSS mix-blend-mode
// и canvas globalCompositeOperation, поэтому список один на холст и на SVG.
var SceneBlends = map[string]bool{
	"multiply": true, "screen": true, "overlay": true, "darken": true, "lighten": true,
	"color-dodge": true, "color-burn": true, "hard-light": true, "soft-light": true,
	"difference": true, "exclusion": true, "hue": true, "saturation": true,
	"color": true, "luminosity": true,
}

// Булевы операции над фигурами (зеркало BOOL_OPS на фронте).
const (
	BoolUnion     = "union"
	BoolSubtract  = "subtract"
	BoolIntersect = "intersect"
	BoolExclude   = "exclude"
)

var sceneBoolOps = map[string]bool{
	BoolUnion: true, BoolSubtract: true, BoolIntersect: true, BoolExclude: true,
}

// hexColor — произвольный цвет из палитры клиента. Проверка обязательна: цвет
// уезжает в атрибут SVG, и любая строка оттуда стала бы дырой для инъекции.
var hexColor = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// sceneHex — цвет ключа для экспорта; неизвестный ключ — «чернила».
func sceneHex(key string) string {
	if hexColor.MatchString(key) {
		return strings.ToLower(key)
	}
	if c, ok := SceneColors[key]; ok {
		return c
	}
	return SceneColors["ink"]
}

// EmptyScene — сцена новой доски.
func EmptyScene() json.RawMessage {
	return json.RawMessage(`{"version":4,"background":"grid",` +
		`"layers":[{"id":"base","name":"Слой 1","visible":true,"locked":false}],"objects":[]}`)
}

// layerOpacity — прозрачность слоя; поля нет (сцены прежних версий) — «1».
func (l SceneLayer) layerOpacity() float64 {
	if l.Opacity == nil {
		return 1
	}
	return math.Min(1, math.Max(0, *l.Opacity))
}

// ParseScene — разбор сцены; битый/пустой JSON даёт пустую сцену (доска не
// должна ломаться из-за одного плохого сохранения).
func ParseScene(raw json.RawMessage) Scene {
	s := Scene{Version: 4, Background: "grid", Layers: defaultLayers(), Objects: []SceneObject{}}
	if len(raw) == 0 {
		return s
	}
	var parsed Scene
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return s
	}
	if parsed.Objects == nil {
		parsed.Objects = []SceneObject{}
	}
	if !SceneBackgrounds[parsed.Background] {
		parsed.Background = "grid"
	}
	// Сцены без слоёв (первая версия формата) поднимаем до текущей: весь холст
	// уезжает в базовый слой, иначе объекты остались бы «ничьими».
	if len(parsed.Layers) == 0 {
		parsed.Layers = defaultLayers()
	}
	// Нижнему слою обтравляться не по чему — иначе он исчез бы целиком.
	parsed.Layers[0].Clip = false
	known := map[string]bool{}
	for i, l := range parsed.Layers {
		known[l.ID] = true
		if l.Mask != nil && len(l.Mask.Points) < 6 {
			parsed.Layers[i].Mask = nil
		}
		if l.Blend != "" && !SceneBlends[l.Blend] {
			parsed.Layers[i].Blend = ""
		}
	}
	frames := map[string]bool{}
	if parsed.Animation != nil {
		for _, f := range parsed.Animation.Frames {
			frames[f.ID] = true
		}
		if len(frames) == 0 {
			parsed.Animation = nil
		}
	}
	// Булева группа из одного участника смысла не имеет — метку снимаем.
	boolCounts := map[string]int{}
	for _, o := range parsed.Objects {
		if o.Bool != "" {
			boolCounts[o.Bool]++
		}
	}
	fallback := parsed.Layers[0].ID
	for i := range parsed.Objects {
		if !known[parsed.Objects[i].Layer] {
			parsed.Objects[i].Layer = fallback
		}
		if parsed.Objects[i].Bool != "" && boolCounts[parsed.Objects[i].Bool] < 2 {
			parsed.Objects[i].Bool = ""
			parsed.Objects[i].BoolOp = ""
		}
		if parsed.Objects[i].BoolOp != "" && !sceneBoolOps[parsed.Objects[i].BoolOp] {
			parsed.Objects[i].BoolOp = BoolUnion
		}
		// Объект удалённого кадра становится общим, а не пропадает: терять
		// нарисованное нельзя (зеркало normalizeScene на фронте).
		if parsed.Objects[i].Frame != "" && !frames[parsed.Objects[i].Frame] {
			parsed.Objects[i].Frame = ""
		}
	}
	parsed.Version = 4
	return parsed
}

// exportFrame — кадр, который уезжает в файл: первый (как и в растровой
// выгрузке клиента). Пусто — доска не анимирована и показывается целиком.
func (s Scene) exportFrame() string {
	if s.Animation == nil || len(s.Animation.Frames) == 0 {
		return ""
	}
	return s.Animation.Frames[0].ID
}

// layerObjects — объекты слоя в порядке отрисовки, без чужих кадров.
func (s Scene) layerObjects(layerID, frame string) []SceneObject {
	out := make([]SceneObject, 0, len(s.Objects))
	for _, o := range s.Objects {
		if o.Layer != layerID || o.Hidden {
			continue
		}
		if frame != "" && o.Frame != "" && o.Frame != frame {
			continue
		}
		out = append(out, o)
	}
	return out
}

// visibleObjects — объекты в порядке отрисовки: снизу вверх по слоям, скрытые
// слои пропускаются (в выгрузке их быть не должно — их не видно и на холсте).
func (s Scene) visibleObjects() []SceneObject {
	frame := s.exportFrame()
	out := make([]SceneObject, 0, len(s.Objects))
	for _, l := range s.Layers {
		if !l.Visible {
			continue
		}
		out = append(out, s.layerObjects(l.ID, frame)...)
	}
	return out
}

// SceneText — плоский текст надписей, стикеров и обсуждений: пересчитывается
// при каждом сохранении и кладётся в text_content (сквозной поиск и превью).
func SceneText(raw json.RawMessage) string {
	var b strings.Builder
	write := func(t string) {
		t = strings.TrimSpace(t)
		if t == "" {
			return
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(t)
	}
	for _, o := range ParseScene(raw).Objects {
		write(o.Text)
		for _, r := range o.Replies {
			write(r.Text)
		}
	}
	return b.String()
}

// StorageKey — ключ объекта в хранилище из адреса, который положил клиент.
// Клиент хранит в сцене готовый URL «/uploads/<key>» (им же он и рисует), а
// хранилище знает только сам ключ — без этого срезания картинки не находились
// ни при выгрузке в SVG, ни при чистке файлов удалённой доски.
func StorageKey(src string) string {
	return strings.TrimPrefix(strings.TrimPrefix(src, "/uploads/"), "uploads/")
}

// SceneImageKeys — ключи картинок, использованных на доске (чистка файлов при
// удалении доски).
func SceneImageKeys(raw json.RawMessage) []string {
	out := []string{}
	for _, o := range ParseScene(raw).Objects {
		if o.Type == ObjImage && o.Src != "" {
			out = append(out, StorageKey(o.Src))
		}
	}
	return out
}

/*
SceneWithoutImages — сцена без картинок с перечисленными ключами (человек

	убирает файл в разделе «Настройки → Хранилище»). Второе значение — менялось
	ли что-нибудь.

	Правим сырое дерево, а не разобранную Scene: та знает лишь те поля, что
	нужны серверу, и пересборка из неё потеряла бы всё, что кладёт клиент.
*/
func SceneWithoutImages(raw json.RawMessage, keys []string) (json.RawMessage, bool) {
	if len(raw) == 0 || len(keys) == 0 {
		return raw, false
	}
	var root map[string]any
	if json.Unmarshal(raw, &root) != nil {
		return raw, false
	}
	objects, ok := root["objects"].([]any)
	if !ok {
		return raw, false
	}
	drop := make(map[string]bool, len(keys))
	for _, k := range keys {
		drop[k] = true
	}
	kept := make([]any, 0, len(objects))
	for _, item := range objects {
		obj, ok := item.(map[string]any)
		if ok {
			if src, _ := obj["src"].(string); src != "" && drop[StorageKey(src)] {
				continue
			}
		}
		kept = append(kept, item)
	}
	if len(kept) == len(objects) {
		return raw, false
	}
	root["objects"] = kept
	out, err := json.Marshal(root)
	if err != nil {
		return raw, false
	}
	return out, true
}

// TextToScene — сцена из плоского текста (импорт .txt): по надписи на строку.
func TextToScene(text string) json.RawMessage {
	objs := []SceneObject{}
	y := 40.0
	for i, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			y += 20
			continue
		}
		objs = append(objs, SceneObject{
			ID: fmt.Sprintf("t%d", i), Type: ObjText, X: 40, Y: y,
			Text: line, Size: 18, Color: "ink",
		})
		y += 32
	}
	raw, err := json.Marshal(Scene{Version: 1, Background: "grid", Objects: objs})
	if err != nil {
		return EmptyScene()
	}
	return raw
}

// localBounds — рамка объекта БЕЗ учёта поворота (в ней объект и нарисован).
func (o SceneObject) localBounds() (minX, minY, maxX, maxY float64) {
	switch o.Type {
	case ObjVector:
		minX, minY = math.Inf(1), math.Inf(1)
		maxX, maxY = math.Inf(-1), math.Inf(-1)
		for _, n := range o.Nodes {
			// Габарит считаем по узлам и направляющим: кривая за них не
			// выходит, а точный обвод здесь не нужен — это рамка листа.
			for _, p := range [3][2]float64{{n.X, n.Y}, {n.IX, n.IY}, {n.OX, n.OY}} {
				minX, minY = math.Min(minX, p[0]), math.Min(minY, p[1])
				maxX, maxY = math.Max(maxX, p[0]), math.Max(maxY, p[1])
			}
		}
		if math.IsInf(minX, 1) {
			return o.X, o.Y, o.X, o.Y
		}
		return minX, minY, maxX, maxY
	case ObjPath:
		minX, minY = math.Inf(1), math.Inf(1)
		maxX, maxY = math.Inf(-1), math.Inf(-1)
		for i := 0; i+1 < len(o.Points); i += 2 {
			minX, minY = math.Min(minX, o.Points[i]), math.Min(minY, o.Points[i+1])
			maxX, maxY = math.Max(maxX, o.Points[i]), math.Max(maxY, o.Points[i+1])
		}
		if math.IsInf(minX, 1) {
			return o.X, o.Y, o.X, o.Y
		}
		return minX, minY, maxX, maxY
	case ObjLine, ObjArrow:
		return math.Min(o.X, o.X2), math.Min(o.Y, o.Y2), math.Max(o.X, o.X2), math.Max(o.Y, o.Y2)
	case ObjComment:
		return o.X, o.Y, o.X + commentPin, o.Y + commentPin
	case ObjText:
		size := o.Size
		if size == 0 {
			size = 18
		}
		return o.X, o.Y - size, o.X + size*float64(len([]rune(o.Text)))*0.6, o.Y + size*0.4
	default:
		return o.X, o.Y, o.X + o.W, o.Y + o.H
	}
}

func (o SceneObject) center() (float64, float64) {
	minX, minY, maxX, maxY := o.localBounds()
	return (minX + maxX) / 2, (minY + maxY) / 2
}

// aabb — габарит объекта с учётом поворота (рамка сцены и выгрузка).
func (o SceneObject) aabb() (minX, minY, maxX, maxY float64) {
	minX, minY, maxX, maxY = o.localBounds()
	if o.Angle == 0 {
		return
	}
	cx, cy := (minX+maxX)/2, (minY+maxY)/2
	rad := o.Angle * math.Pi / 180
	cos, sin := math.Cos(rad), math.Sin(rad)
	corners := [4][2]float64{{minX, minY}, {maxX, minY}, {maxX, maxY}, {minX, maxY}}
	rx0, ry0 := math.Inf(1), math.Inf(1)
	rx1, ry1 := math.Inf(-1), math.Inf(-1)
	for _, c := range corners {
		dx, dy := c[0]-cx, c[1]-cy
		x, y := cx+dx*cos-dy*sin, cy+dx*sin+dy*cos
		rx0, ry0 = math.Min(rx0, x), math.Min(ry0, y)
		rx1, ry1 = math.Max(rx1, x), math.Max(ry1, y)
	}
	return rx0, ry0, rx1, ry1
}

// bounds — рамка сцены с полями; пустая сцена даёт лист по умолчанию.
func (s Scene) bounds() (minX, minY, maxX, maxY float64) {
	minX, minY = math.Inf(1), math.Inf(1)
	maxX, maxY = math.Inf(-1), math.Inf(-1)
	for _, o := range s.visibleObjects() {
		x0, y0, x1, y1 := o.aabb()
		minX, minY = math.Min(minX, x0), math.Min(minY, y0)
		maxX, maxY = math.Max(maxX, x1), math.Max(maxY, y1)
	}
	if math.IsInf(minX, 1) {
		return 0, 0, 1200, 800
	}
	const pad = 40
	return minX - pad, minY - pad, maxX + pad, maxY + pad
}
