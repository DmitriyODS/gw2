package domain

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

func base64Encode(data []byte) string { return base64.StdEncoding.EncodeToString(data) }

// Scene — содержимое доски: плоский список объектов холста. Клиент рисует их
// сам (Canvas/SVG), сервер знает ровно столько, сколько нужно ему самому:
// извлечь текст надписей для поиска и построить SVG при экспорте.
//
// Цвет объекта — КЛЮЧ палитры (ink/red/blue…), а не hex: фронт разворачивает
// ключ в токен --tag-*, поэтому доска остаётся в теме приложения и в тёмном
// режиме. Конкретные значения нужны только серверу для SVG-экспорта (файл
// уезжает наружу, где токенов нет) — sceneHex ниже.
type Scene struct {
	Version    int           `json:"version"`
	Background string        `json:"background"`
	Layers     []SceneLayer  `json:"layers"`
	Objects    []SceneObject `json:"objects"`
}

// SceneLayer — слой холста: порядок в срезе задаёт порядок отрисовки (первый —
// нижний), Visible скрывает слой целиком, Locked запрещает правку в клиенте.
type SceneLayer struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Visible bool   `json:"visible"`
	Locked  bool   `json:"locked"`
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
	// Color/Fill — ключи палитры (Fill "" — без заливки).
	Color   string  `json:"color,omitempty"`
	Fill    string  `json:"fill,omitempty"`
	Width   float64 `json:"width,omitempty"`
	Opacity float64 `json:"opacity,omitempty"`
	Text    string  `json:"text,omitempty"`
	Size    float64 `json:"size,omitempty"`
	// Src — адрес картинки, каким его положил клиент (/uploads/<key>).
	Src string `json:"src,omitempty"`
	// Layer — слой объекта, Group — id группы (объекты группы двигаются вместе).
	Layer string `json:"layer,omitempty"`
	Group string `json:"group,omitempty"`
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
	ObjPath    = "path"    // свободное перо/маркер
	ObjLine    = "line"    // прямая
	ObjArrow   = "arrow"   // стрелка
	ObjRect    = "rect"    // прямоугольник
	ObjEllipse = "ellipse" // эллипс
	ObjDiamond = "diamond" // ромб
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
// тегов задач (front/src/utils/taskColors.js) плюс нейтральные «чернила».
var SceneColors = map[string]string{
	"ink":    "#1f2430",
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

// sceneHex — цвет ключа для экспорта; неизвестный ключ — «чернила».
func sceneHex(key string) string {
	if c, ok := SceneColors[key]; ok {
		return c
	}
	return SceneColors["ink"]
}

// EmptyScene — сцена новой доски.
func EmptyScene() json.RawMessage {
	return json.RawMessage(`{"version":2,"background":"grid",` +
		`"layers":[{"id":"base","name":"Слой 1","visible":true,"locked":false}],"objects":[]}`)
}

// ParseScene — разбор сцены; битый/пустой JSON даёт пустую сцену (доска не
// должна ломаться из-за одного плохого сохранения).
func ParseScene(raw json.RawMessage) Scene {
	s := Scene{Version: 2, Background: "grid", Layers: defaultLayers(), Objects: []SceneObject{}}
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
	known := map[string]bool{}
	for _, l := range parsed.Layers {
		known[l.ID] = true
	}
	fallback := parsed.Layers[0].ID
	for i := range parsed.Objects {
		if !known[parsed.Objects[i].Layer] {
			parsed.Objects[i].Layer = fallback
		}
	}
	parsed.Version = 2
	return parsed
}

// visibleObjects — объекты в порядке отрисовки: снизу вверх по слоям, скрытые
// слои пропускаются (в выгрузке их быть не должно — их не видно и на холсте).
func (s Scene) visibleObjects() []SceneObject {
	out := make([]SceneObject, 0, len(s.Objects))
	for _, l := range s.Layers {
		if !l.Visible {
			continue
		}
		for _, o := range s.Objects {
			if o.Layer == l.ID {
				out = append(out, o)
			}
		}
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

/* SceneWithoutImages — сцена без картинок с перечисленными ключами (человек
   убирает файл в разделе «Настройки → Хранилище»). Второе значение — менялось
   ли что-нибудь.

   Правим сырое дерево, а не разобранную Scene: та знает лишь те поля, что
   нужны серверу, и пересборка из неё потеряла бы всё, что кладёт клиент. */
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

// bounds — рамка сцены с полями; пустая сцена даёт лист по умолчанию.
func (s Scene) bounds() (minX, minY, maxX, maxY float64) {
	minX, minY = math.Inf(1), math.Inf(1)
	maxX, maxY = math.Inf(-1), math.Inf(-1)
	grow := func(x, y float64) {
		minX, minY = math.Min(minX, x), math.Min(minY, y)
		maxX, maxY = math.Max(maxX, x), math.Max(maxY, y)
	}
	for _, o := range s.visibleObjects() {
		switch o.Type {
		case ObjPath:
			for i := 0; i+1 < len(o.Points); i += 2 {
				grow(o.Points[i], o.Points[i+1])
			}
		case ObjLine, ObjArrow:
			grow(o.X, o.Y)
			grow(o.X2, o.Y2)
		case ObjComment:
			grow(o.X, o.Y)
			grow(o.X+commentPin, o.Y+commentPin)
		case ObjText:
			size := o.Size
			if size == 0 {
				size = 18
			}
			grow(o.X, o.Y-size)
			grow(o.X+size*float64(len([]rune(o.Text)))*0.6, o.Y+size*0.4)
		default:
			grow(o.X, o.Y)
			grow(o.X+o.W, o.Y+o.H)
		}
	}
	if math.IsInf(minX, 1) {
		return 0, 0, 1200, 800
	}
	const pad = 40
	return minX - pad, minY - pad, maxX + pad, maxY + pad
}

// SceneSVG — векторный экспорт доски (файл уезжает наружу, поэтому здесь
// конкретные цвета вместо токенов — как в шаблонах писем mailsvc).
// resolveImage — читатель байтов картинки по ключу для встраивания data-URI;
// nil или ошибка чтения → картинка пропускается.
func SceneSVG(raw json.RawMessage, resolveImage func(key string) (mime string, data []byte, err error)) []byte {
	s := ParseScene(raw)
	minX, minY, maxX, maxY := s.bounds()
	w, h := maxX-minX, maxY-minY

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" `+
		`viewBox="%.1f %.1f %.1f %.1f" width="%.0f" height="%.0f">`, minX, minY, w, h, w, h)
	b.WriteString(`<defs><marker id="ah" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" ` +
		`orient="auto-start-reverse"><path d="M0,0 L10,5 L0,10 z" fill="context-stroke"/></marker></defs>`)
	fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="#ffffff"/>`, minX, minY, w, h)

	for _, o := range s.visibleObjects() {
		writeSVGObject(&b, o, resolveImage)
	}
	b.WriteString(`</svg>`)
	return []byte(b.String())
}

func writeSVGObject(b *strings.Builder, o SceneObject, resolveImage func(string) (string, []byte, error)) {
	stroke := sceneHex(o.Color)
	width := o.Width
	if width == 0 {
		width = 3
	}
	opacity := o.Opacity
	if opacity == 0 {
		opacity = 1
	}
	fill := "none"
	if o.Fill != "" {
		fill = sceneHex(o.Fill)
	}

	switch o.Type {
	case ObjPath:
		if len(o.Points) < 4 {
			return
		}
		var d strings.Builder
		for i := 0; i+1 < len(o.Points); i += 2 {
			cmd := "L"
			if i == 0 {
				cmd = "M"
			}
			fmt.Fprintf(&d, "%s%.1f %.1f ", cmd, o.Points[i], o.Points[i+1])
		}
		fmt.Fprintf(b, `<path d="%s" fill="none" stroke="%s" stroke-width="%.1f" stroke-opacity="%.2f" `+
			`stroke-linecap="round" stroke-linejoin="round"/>`, strings.TrimSpace(d.String()), stroke, width, opacity)
	case ObjLine, ObjArrow:
		marker := ""
		if o.Type == ObjArrow {
			marker = ` marker-end="url(#ah)"`
		}
		fmt.Fprintf(b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.1f" `+
			`stroke-linecap="round"%s/>`, o.X, o.Y, o.X2, o.Y2, stroke, width, marker)
	case ObjRect:
		fmt.Fprintf(b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="8" fill="%s" stroke="%s" `+
			`stroke-width="%.1f"/>`, o.X, o.Y, o.W, o.H, fill, stroke, width)
	case ObjEllipse:
		fmt.Fprintf(b, `<ellipse cx="%.1f" cy="%.1f" rx="%.1f" ry="%.1f" fill="%s" stroke="%s" `+
			`stroke-width="%.1f"/>`, o.X+o.W/2, o.Y+o.H/2, o.W/2, o.H/2, fill, stroke, width)
	case ObjDiamond:
		fmt.Fprintf(b, `<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s" stroke="%s" `+
			`stroke-width="%.1f"/>`, o.X+o.W/2, o.Y, o.X+o.W, o.Y+o.H/2, o.X+o.W/2, o.Y+o.H, o.X, o.Y+o.H/2,
			fill, stroke, width)
	case ObjSticky:
		note := sceneHex(o.Color)
		fmt.Fprintf(b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="6" fill="%s" fill-opacity="0.35" `+
			`stroke="%s" stroke-width="1"/>`, o.X, o.Y, o.W, o.H, note, note)
		writeSVGText(b, o.Text, o.X+12, o.Y+26, 16, sceneHex("ink"))
	case ObjText:
		size := o.Size
		if size == 0 {
			size = 18
		}
		writeSVGText(b, o.Text, o.X, o.Y, size, stroke)
	case ObjComment:
		tint := sceneHex(o.Color)
		if o.Resolved {
			tint = sceneHex("green")
		}
		r := float64(commentPin) / 2
		fmt.Fprintf(b, `<circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s" fill-opacity="%.2f"/>`,
			o.X+r, o.Y+r, r, tint, map[bool]float64{true: 0.55, false: 1}[o.Resolved])
		writeSVGText(b, strconv.Itoa(1+len(o.Replies)), o.X+r-4, o.Y+r+5, 14, sceneHex("chalk"))
	case ObjImage:
		if o.Src == "" || resolveImage == nil {
			return
		}
		mime, data, err := resolveImage(StorageKey(o.Src))
		if err != nil || len(data) == 0 {
			return
		}
		fmt.Fprintf(b, `<image x="%.1f" y="%.1f" width="%.1f" height="%.1f" xlink:href="data:%s;base64,%s"/>`,
			o.X, o.Y, o.W, o.H, mime, base64Encode(data))
	}
}

// writeSVGText — многострочная надпись (SVG сам переносы не делает).
func writeSVGText(b *strings.Builder, text string, x, y, size float64, color string) {
	lines := strings.Split(text, "\n")
	fmt.Fprintf(b, `<text x="%.1f" y="%.1f" font-family="Inter, Arial, sans-serif" font-size="%.1f" fill="%s">`,
		x, y, size, color)
	for i, line := range lines {
		dy := 0.0
		if i > 0 {
			dy = size * 1.35
		}
		fmt.Fprintf(b, `<tspan x="%.1f" dy="%.1f">%s</tspan>`, x, dy, escapeXML(line))
	}
	b.WriteString(`</text>`)
}

func escapeXML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}
