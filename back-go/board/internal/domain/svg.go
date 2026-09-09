package domain

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

/* Векторная выгрузка доски. Файл уезжает наружу, где ни темы приложения, ни
   холста нет, поэтому здесь конкретные цвета вместо токенов — как в шаблонах
   писем mailsvc.

   Возможности сцены ложатся на SVG так:
     • слой            — <g> с opacity и mix-blend-mode;
     • эффекты         — <filter> (размытие, коррекции, тень);
     • градиент        — <linearGradient>/<radialGradient> в defs;
     • стирание, маска слоя — <mask> ПО ЯРКОСТИ (белое видно, чёрное скрыто);
     • обтравка и объект-маска — <mask> ПО АЛЬФЕ (mask-type:alpha): яркостью их
       не выразить, содержимое цветное;
     • булевы операции — объединение и исключение одним контуром с fill-rule,
       вычитание и пересечение — цепочкой масок (ровно как композит на холсте).

   Всё, что уезжает в атрибут, проходит проверку: цвет — по регэкспу, числа
   форматируются сами, текст экранируется. */

// svgArea — рамка листа: маски задаются в тех же координатах, что и рисунок
// (userSpaceOnUse), иначе SVG считает их долями рамки объекта.
type svgArea struct{ x, y, w, h float64 }

func (a svgArea) attrs() string {
	return fmt.Sprintf(`maskUnits="userSpaceOnUse" x="%.1f" y="%.1f" width="%.1f" height="%.1f"`, a.x, a.y, a.w, a.h)
}

// svgDoc — состояние сборки: defs копятся отдельно от тела рисунка.
type svgDoc struct {
	defs    strings.Builder
	area    svgArea
	seq     int
	scene   Scene
	frame   string
	resolve func(string) (string, []byte, error)
}

func (d *svgDoc) id(prefix string) string {
	d.seq++
	return fmt.Sprintf("%s%d", prefix, d.seq)
}

// SceneSVG — векторный экспорт доски; resolveImage читает байты картинки по
// ключу для встраивания data-URI (nil или ошибка — картинка пропускается).
func SceneSVG(raw json.RawMessage, resolveImage func(key string) (mime string, data []byte, err error)) []byte {
	s := ParseScene(raw)
	minX, minY, maxX, maxY := s.bounds()
	w, h := maxX-minX, maxY-minY
	doc := &svgDoc{
		area:    svgArea{x: minX, y: minY, w: w, h: h},
		scene:   s,
		frame:   s.exportFrame(),
		resolve: resolveImage,
	}

	var body strings.Builder
	// clipBase — слой, по которому обтравливаются идущие над ним clip-слои.
	var clipBase *SceneLayer

	for i := range s.Layers {
		layer := s.Layers[i]
		if !layer.Visible {
			if !layer.Clip {
				clipBase = nil
			}
			continue
		}
		objects := s.layerObjects(layer.ID, doc.frame)

		attrs := ""
		if op := layer.layerOpacity(); op < 1 {
			attrs += fmt.Sprintf(` opacity="%.2f"`, op)
		}
		if SceneBlends[layer.Blend] {
			attrs += fmt.Sprintf(` style="mix-blend-mode:%s"`, layer.Blend)
		}
		if id := doc.filterFor(layer.Effects); id != "" {
			attrs += fmt.Sprintf(` filter="url(#%s)"`, id)
		}
		if id := doc.contentMask(layer, objects); id != "" {
			attrs += fmt.Sprintf(` mask="url(#%s)"`, id)
		}

		var group strings.Builder
		fmt.Fprintf(&group, `<g%s>`, attrs)
		doc.writeSections(&group, objects)
		group.WriteString(`</g>`)

		if layer.Clip && clipBase != nil {
			id := doc.alphaMask(s.layerObjects(clipBase.ID, doc.frame))
			fmt.Fprintf(&body, `<g mask="url(#%s)">%s</g>`, id, group.String())
			continue
		}
		body.WriteString(group.String())
		base := layer
		clipBase = &base
	}

	var out strings.Builder
	fmt.Fprintf(&out, `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" `+
		`viewBox="%.1f %.1f %.1f %.1f" width="%.0f" height="%.0f">`, minX, minY, w, h, w, h)
	out.WriteString(`<defs><marker id="ah" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" ` +
		`orient="auto-start-reverse"><path d="M0,0 L10,5 L0,10 z" fill="context-stroke"/></marker>`)
	out.WriteString(doc.defs.String())
	out.WriteString(`</defs>`)
	fmt.Fprintf(&out, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="#ffffff"/>`, minX, minY, w, h)
	out.WriteString(body.String())
	out.WriteString(`</svg>`)
	return []byte(out.String())
}

/*
writeSections — содержимое слоя с учётом объектов-масок: маска показывает

	всё, что лежит НАД ней в слое, только внутри своей формы — до следующей
	маски (то же правило, что на холсте).
*/
func (d *svgDoc) writeSections(b *strings.Builder, objects []SceneObject) {
	section := make([]SceneObject, 0, len(objects))
	var mask *SceneObject

	flush := func() {
		if len(section) == 0 && mask == nil {
			return
		}
		if mask == nil {
			d.writeItems(b, section)
			return
		}
		id := d.alphaMask([]SceneObject{*mask})
		fmt.Fprintf(b, `<g mask="url(#%s)">`, id)
		d.writeItems(b, section)
		b.WriteString(`</g>`)
	}

	for _, o := range objects {
		if o.MaskObject {
			flush()
			copy := o
			mask = &copy
			section = section[:0]
			continue
		}
		section = append(section, o)
	}
	flush()
}

// writeItems — объекты по порядку; булева группа рисуется целиком на месте базы.
func (d *svgDoc) writeItems(b *strings.Builder, objects []SceneObject) {
	done := map[string]bool{}
	for _, o := range objects {
		if done[o.ID] {
			continue
		}
		if o.Erase {
			continue // стирание живёт в маске, самим объектом не рисуется
		}
		if o.Bool != "" {
			members := make([]SceneObject, 0, 4)
			for _, m := range objects {
				if m.Bool == o.Bool {
					members = append(members, m)
					done[m.ID] = true
				}
			}
			d.writeBoolGroup(b, members)
			continue
		}
		d.writeObject(b, o)
	}
}

/*
writeBoolGroup — булева группа. Объединение и исключение выражаются ОДНИМ

	контуром с разным fill-rule, вычитание и пересечение — масками по альфе:
	маска строится по каждому участнику отдельно, поэтому результат совпадает с
	последовательным композитом на холсте.
*/
func (d *svgDoc) writeBoolGroup(b *strings.Builder, members []SceneObject) {
	if len(members) == 0 {
		return
	}
	base := members[0]
	rest := members[1:]

	// Группа однородна по операции? Тогда её можно выразить одним контуром.
	op := BoolUnion
	if len(rest) > 0 {
		op = rest[0].BoolOp
		for _, m := range rest {
			if m.BoolOp != op {
				op = ""
				break
			}
		}
	}

	if op == BoolUnion || op == BoolExclude {
		rule := "nonzero"
		if op == BoolExclude {
			rule = "evenodd"
		}
		var d2 strings.Builder
		for _, m := range members {
			d2.WriteString(shapePathData(m))
			d2.WriteString(" ")
		}
		style := d.paintAttrs(base, true)
		fmt.Fprintf(b, `<path d="%s" fill-rule="%s"%s/>`, strings.TrimSpace(d2.String()), rule, style)
		return
	}

	// Разнородная или вычитающая группа: оборачиваем базу в цепочку масок.
	opens := 0
	for _, m := range rest {
		switch m.BoolOp {
		case BoolSubtract:
			id := d.shapeMask(m, false)
			fmt.Fprintf(b, `<g mask="url(#%s)">`, id)
			opens++
		case BoolIntersect:
			id := d.shapeMask(m, true)
			fmt.Fprintf(b, `<g mask="url(#%s)">`, id)
			opens++
		}
	}
	// Объединяемые участники рисуются вместе с базой — стилем базы.
	var union strings.Builder
	union.WriteString(shapePathData(base))
	for _, m := range rest {
		if m.BoolOp == BoolUnion || m.BoolOp == BoolExclude {
			union.WriteString(" ")
			union.WriteString(shapePathData(m))
		}
	}
	fmt.Fprintf(b, `<path d="%s"%s/>`, strings.TrimSpace(union.String()), d.paintAttrs(base, true))
	for i := 0; i < opens; i++ {
		b.WriteString(`</g>`)
	}
}

/*
shapeMask — маска из формы одного объекта: keep=true оставляет только то,

	что внутри формы (пересечение), keep=false — вырезает её (вычитание).
*/
func (d *svgDoc) shapeMask(o SceneObject, keep bool) string {
	id := d.id("bm")
	ground, shape := "#ffffff", "#000000"
	if keep {
		ground, shape = "#000000", "#ffffff"
	}
	fmt.Fprintf(&d.defs, `<mask id="%s" %s>`, id, d.area.attrs())
	fmt.Fprintf(&d.defs, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s"/>`,
		d.area.x, d.area.y, d.area.w, d.area.h, ground)
	fmt.Fprintf(&d.defs, `<path d="%s" fill="%s"%s/>`, shapePathData(o), shape, rotateAttr(o))
	d.defs.WriteString(`</mask>`)
	return id
}

/*
contentMask — маска слоя по яркости: белое видно, чёрное скрыто. Нужна,

	когда в слое есть стирающие штрихи или задана многоугольная маска.
*/
func (d *svgDoc) contentMask(layer SceneLayer, objects []SceneObject) string {
	erasers := make([]SceneObject, 0)
	for _, o := range objects {
		if o.Erase {
			erasers = append(erasers, o)
		}
	}
	if len(erasers) == 0 && layer.Mask == nil {
		return ""
	}
	id := d.id("lm")
	fmt.Fprintf(&d.defs, `<mask id="%s" %s>`, id, d.area.attrs())
	ground, cut := "#ffffff", "#000000"
	if layer.Mask != nil && !layer.Mask.Invert {
		// Видно только внутри многоугольника — лист по умолчанию чёрный.
		ground, cut = "#000000", "#ffffff"
	}
	fmt.Fprintf(&d.defs, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s"/>`,
		d.area.x, d.area.y, d.area.w, d.area.h, ground)
	if layer.Mask != nil {
		fmt.Fprintf(&d.defs, `<polygon points="%s" fill="%s"/>`, pointsAttr(layer.Mask.Points), cut)
	}
	for _, o := range erasers {
		writeErasePath(&d.defs, o)
	}
	d.defs.WriteString(`</mask>`)
	return id
}

/*
alphaMask — маска из содержимого: берётся его АЛЬФА (mask-type:alpha). Так

	обтравленный слой и объект-маска показывают то, что над ними, ровно там, где
	их форма непрозрачна.
*/
func (d *svgDoc) alphaMask(objects []SceneObject) string {
	id := d.id("am")
	fmt.Fprintf(&d.defs, `<mask id="%s" %s style="mask-type:alpha">`, id, d.area.attrs())
	for _, o := range objects {
		if o.Erase {
			continue
		}
		clean := o
		clean.MaskObject = false
		d.writeObject(&d.defs, clean)
	}
	d.defs.WriteString(`</mask>`)
	return id
}

// writeErasePath — стирающий штрих внутри маски: чёрным по белому.
func writeErasePath(b *strings.Builder, o SceneObject) {
	d := pathData(o)
	if d == "" {
		return
	}
	if o.Filled {
		fmt.Fprintf(b, `<path d="%s" fill="#000000"/>`, d)
		return
	}
	width := o.Width
	if width == 0 {
		width = 3
	}
	fmt.Fprintf(b, `<path d="%s" fill="none" stroke="#000000" stroke-width="%.1f" `+
		`stroke-linecap="round" stroke-linejoin="round"/>`, d, width)
}

// ── Заливки, градиенты и эффекты ─────────────────────────────────

// gradientFor — id градиента в defs; пустая строка, если градиента нет.
func (d *svgDoc) gradientFor(o SceneObject) string {
	g := o.Gradient
	if g == nil || len(g.Stops) < 2 {
		return ""
	}
	id := d.id("gr")
	x0, y0, x1, y1, minX, minY, maxX, maxY := 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0
	minX, minY, maxX, maxY = o.localBounds()
	cx, cy := (minX+maxX)/2, (minY+maxY)/2
	w, h := maxX-minX, maxY-minY

	if g.Type == "radial" {
		r := math.Max(w, h) / 2
		if r == 0 {
			r = 1
		}
		fmt.Fprintf(&d.defs, `<radialGradient id="%s" gradientUnits="userSpaceOnUse" cx="%.1f" cy="%.1f" r="%.1f">`,
			id, cx, cy, r)
	} else {
		// Угол как в CSS: 0° — снизу вверх, 90° — слева направо.
		rad := (g.Angle - 90) * math.Pi / 180
		half := (math.Abs(w*math.Cos(rad)) + math.Abs(h*math.Sin(rad))) / 2
		if half == 0 {
			half = 1
		}
		x0, y0 = cx-math.Cos(rad)*half, cy-math.Sin(rad)*half
		x1, y1 = cx+math.Cos(rad)*half, cy+math.Sin(rad)*half
		fmt.Fprintf(&d.defs, `<linearGradient id="%s" gradientUnits="userSpaceOnUse" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f">`,
			id, x0, y0, x1, y1)
	}
	for _, st := range g.Stops {
		at := math.Min(1, math.Max(0, st.At))
		fmt.Fprintf(&d.defs, `<stop offset="%.3f" stop-color="%s"/>`, at, sceneHex(st.Color))
	}
	if g.Type == "radial" {
		d.defs.WriteString(`</radialGradient>`)
	} else {
		d.defs.WriteString(`</linearGradient>`)
	}
	return id
}

func effectValue(v *float64, def float64) float64 {
	if v == nil {
		return def
	}
	return *v
}

/*
filterFor — id фильтра коррекций в defs. Порядок примитивов повторяет

	порядок CSS-функций на холсте (blur → brightness → contrast → saturate →
	hue-rotate → grayscale → sepia), иначе картинка в файле отличалась бы от
	того, что человек видел.
*/
func (d *svgDoc) filterFor(e *SceneEffects) string {
	if e == nil {
		return ""
	}
	brightness := effectValue(e.Brightness, 1)
	contrast := effectValue(e.Contrast, 1)
	saturate := effectValue(e.Saturate, 1)
	if e.Blur == 0 && e.Hue == 0 && e.Grayscale == 0 && e.Sepia == 0 &&
		brightness == 1 && contrast == 1 && saturate == 1 && e.Shadow == nil {
		return ""
	}
	id := d.id("fx")
	// Фильтру нужен запас поля: размытие и тень выходят за рамку объекта.
	fmt.Fprintf(&d.defs, `<filter id="%s" x="-30%%" y="-30%%" width="160%%" height="160%%">`, id)
	if e.Blur > 0 {
		fmt.Fprintf(&d.defs, `<feGaussianBlur stdDeviation="%.2f"/>`, e.Blur/2)
	}
	if brightness != 1 || contrast != 1 {
		// Яркость — множитель, контраст — наклон с сдвигом относительно 0.5.
		intercept := -(contrast-1)/2 + 0
		for _, ch := range []string{"R", "G", "B"} {
			fmt.Fprintf(&d.defs, `<feComponentTransfer><feFunc%s type="linear" slope="%.3f" intercept="%.3f"/></feComponentTransfer>`,
				ch, brightness*contrast, intercept)
		}
	}
	if saturate != 1 {
		fmt.Fprintf(&d.defs, `<feColorMatrix type="saturate" values="%.3f"/>`, saturate)
	}
	if e.Hue != 0 {
		fmt.Fprintf(&d.defs, `<feColorMatrix type="hueRotate" values="%.1f"/>`, e.Hue)
	}
	if e.Grayscale > 0 {
		fmt.Fprintf(&d.defs, `<feColorMatrix type="saturate" values="%.3f"/>`, 1-math.Min(1, e.Grayscale))
	}
	if e.Sepia > 0 {
		k := math.Min(1, e.Sepia)
		// Матрица сепии из CSS Filter Effects, смешанная с исходной по силе.
		fmt.Fprintf(&d.defs, `<feColorMatrix type="matrix" values="%s"/>`, sepiaMatrix(k))
	}
	if e.Shadow != nil {
		fmt.Fprintf(&d.defs, `<feDropShadow dx="%.1f" dy="%.1f" stdDeviation="%.2f" flood-color="%s" flood-opacity="%.2f"/>`,
			e.Shadow.X, e.Shadow.Y, e.Shadow.Blur/2, sceneHex(e.Shadow.Color), math.Min(1, math.Max(0, e.Shadow.Opacity)))
	}
	d.defs.WriteString(`</filter>`)
	return id
}

// sepiaMatrix — матрица сепии, смешанная с единичной по силе k.
func sepiaMatrix(k float64) string {
	base := [][]float64{
		{0.393, 0.769, 0.189},
		{0.349, 0.686, 0.168},
		{0.272, 0.534, 0.131},
	}
	identity := [][]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}
	var b strings.Builder
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			fmt.Fprintf(&b, "%.4f ", identity[row][col]*(1-k)+base[row][col]*k)
		}
		b.WriteString("0 0 ")
	}
	b.WriteString("0 0 0 1 0")
	return strings.TrimSpace(b.String())
}

// paintAttrs — атрибуты заливки и обводки объекта (общий код фигур и булевых).
func (d *svgDoc) paintAttrs(o SceneObject, forceFill bool) string {
	stroke := sceneHex(o.Color)
	width := o.Width
	if width == 0 && o.Type != ObjPath && o.Type != ObjVector {
		width = 3
	}
	opacity := o.Opacity
	if opacity == 0 {
		opacity = 1
	}
	fill := "none"
	fillOpacity := 0.0
	if id := d.gradientFor(o); id != "" {
		fill = fmt.Sprintf("url(#%s)", id)
		fillOpacity = 1
	} else if o.Fill != "" {
		fill = sceneHex(o.Fill)
		fillOpacity = fillAlpha(o)
	} else if forceFill {
		fill = stroke
		fillOpacity = 1
	}
	attrs := fmt.Sprintf(` fill="%s" fill-opacity="%.2f" stroke="%s" stroke-width="%.1f" stroke-opacity="%.2f"`,
		fill, fillOpacity*opacity, stroke, width, opacity)
	if id := d.filterFor(o.Effects); id != "" {
		attrs += fmt.Sprintf(` filter="url(#%s)"`, id)
	}
	return attrs
}

// rotateAttr — поворот объекта вокруг центра его рамки.
func rotateAttr(o SceneObject) string {
	if o.Angle == 0 {
		return ""
	}
	cx, cy := o.center()
	return fmt.Sprintf(` transform="rotate(%.1f %.1f %.1f)"`, o.Angle, cx, cy)
}

// ── Контуры ──────────────────────────────────────────────────────

// pointsAttr — список точек для polygon/polyline.
func pointsAttr(points []float64) string {
	var b strings.Builder
	for i := 0; i+1 < len(points); i += 2 {
		if i > 0 {
			b.WriteString(" ")
		}
		fmt.Fprintf(&b, "%.1f,%.1f", points[i], points[i+1])
	}
	return b.String()
}

/*
pathData — контур свободной кривой. Smooth даёт квадратичные сегменты через

	середины отрезков — ЗЕРКАЛО pathShape на фронте: иначе выгрузка кисти
	отличалась бы от того, что человек видел на холсте.
*/
func pathData(o SceneObject) string {
	pts := o.Points
	if len(pts) < 4 {
		return ""
	}
	var d strings.Builder
	fmt.Fprintf(&d, "M%.1f %.1f ", pts[0], pts[1])
	if o.Smooth && len(pts) >= 6 {
		for i := 2; i+3 < len(pts); i += 2 {
			mx := (pts[i] + pts[i+2]) / 2
			my := (pts[i+1] + pts[i+3]) / 2
			fmt.Fprintf(&d, "Q%.1f %.1f %.1f %.1f ", pts[i], pts[i+1], mx, my)
		}
		fmt.Fprintf(&d, "L%.1f %.1f ", pts[len(pts)-2], pts[len(pts)-1])
	} else {
		for i := 2; i+1 < len(pts); i += 2 {
			fmt.Fprintf(&d, "L%.1f %.1f ", pts[i], pts[i+1])
		}
	}
	if o.Closed {
		d.WriteString("Z")
	}
	return strings.TrimSpace(d.String())
}

// vectorPathData — контур узлов Безье (зеркало vectorShape на фронте).
func vectorPathData(o SceneObject) string {
	if len(o.Nodes) < 2 {
		return ""
	}
	var d strings.Builder
	fmt.Fprintf(&d, "M%.1f %.1f ", o.Nodes[0].X, o.Nodes[0].Y)
	for i := 0; i+1 < len(o.Nodes); i++ {
		a, b := o.Nodes[i], o.Nodes[i+1]
		fmt.Fprintf(&d, "C%.1f %.1f %.1f %.1f %.1f %.1f ", a.OX, a.OY, b.IX, b.IY, b.X, b.Y)
	}
	if o.Closed {
		a, b := o.Nodes[len(o.Nodes)-1], o.Nodes[0]
		fmt.Fprintf(&d, "C%.1f %.1f %.1f %.1f %.1f %.1f Z", a.OX, a.OY, b.IX, b.IY, b.X, b.Y)
	}
	return strings.TrimSpace(d.String())
}

// polygonPoints — вершины много-/звезды (зеркало polygonPoints на фронте).
func polygonPoints(o SceneObject) []float64 {
	sides := o.Sides
	if sides < 3 {
		sides = 5
	}
	if sides > 24 {
		sides = 24
	}
	inner := o.Inner
	if inner <= 0 || inner > 0.9 {
		inner = 0.5
	}
	cx, cy := o.X+o.W/2, o.Y+o.H/2
	rx, ry := o.W/2, o.H/2
	count := sides
	if o.Star {
		count = sides * 2
	}
	out := make([]float64, 0, count*2)
	for i := 0; i < count; i++ {
		angle := 2*math.Pi*float64(i)/float64(count) - math.Pi/2
		k := 1.0
		if o.Star && i%2 == 1 {
			k = inner
		}
		out = append(out, cx+math.Cos(angle)*rx*k, cy+math.Sin(angle)*ry*k)
	}
	return out
}

// polygonPathData — полигон как команды пути (нужно булевым операциям).
func polygonPathData(points []float64) string {
	if len(points) < 4 {
		return ""
	}
	var d strings.Builder
	fmt.Fprintf(&d, "M%.1f %.1f ", points[0], points[1])
	for i := 2; i+1 < len(points); i += 2 {
		fmt.Fprintf(&d, "L%.1f %.1f ", points[i], points[i+1])
	}
	d.WriteString("Z")
	return d.String()
}

/*
shapePathData — форма ЛЮБОГО объекта командами пути: булевым операциям нужен

	общий язык, а <rect> и <ellipse> в один контур не сложить.
*/
func shapePathData(o SceneObject) string {
	switch o.Type {
	case ObjVector:
		return vectorPathData(o)
	case ObjPath:
		return pathData(o)
	case ObjEllipse:
		rx, ry := o.W/2, o.H/2
		cx, cy := o.X+rx, o.Y+ry
		return fmt.Sprintf("M%.1f %.1f A%.1f %.1f 0 1 0 %.1f %.1f A%.1f %.1f 0 1 0 %.1f %.1f Z",
			cx-rx, cy, rx, ry, cx+rx, cy, rx, ry, cx-rx, cy)
	case ObjDiamond:
		return polygonPathData([]float64{
			o.X + o.W/2, o.Y, o.X + o.W, o.Y + o.H/2, o.X + o.W/2, o.Y + o.H, o.X, o.Y + o.H/2,
		})
	case ObjPolygon:
		return polygonPathData(polygonPoints(o))
	case ObjLine, ObjArrow:
		return fmt.Sprintf("M%.1f %.1f L%.1f %.1f", o.X, o.Y, o.X2, o.Y2)
	default:
		return polygonPathData([]float64{o.X, o.Y, o.X + o.W, o.Y, o.X + o.W, o.Y + o.H, o.X, o.Y + o.H})
	}
}

// ── Объекты ──────────────────────────────────────────────────────

func (d *svgDoc) writeObject(b *strings.Builder, o SceneObject) {
	rotate := rotateAttr(o)
	if rotate != "" {
		fmt.Fprintf(b, `<g%s>`, rotate)
		plain := o
		plain.Angle = 0
		d.writeObject(b, plain)
		b.WriteString(`</g>`)
		return
	}

	stroke := sceneHex(o.Color)
	width := o.Width
	if width == 0 && o.Type != ObjPath && o.Type != ObjVector {
		width = 3
	}
	opacity := o.Opacity
	if opacity == 0 {
		opacity = 1
	}

	switch o.Type {
	case ObjPath, ObjVector:
		data := pathData(o)
		if o.Type == ObjVector {
			data = vectorPathData(o)
		}
		if data == "" {
			return
		}
		if o.Filled || o.Gradient != nil {
			paint := "none"
			alpha := 0.35
			if id := d.gradientFor(o); id != "" {
				paint = fmt.Sprintf("url(#%s)", id)
				alpha = 1
			} else {
				key := o.Fill
				if key == "" {
					key = o.Color
				}
				paint = sceneHex(key)
				if o.Solid {
					alpha = 1
				}
			}
			fmt.Fprintf(b, `<path d="%s" fill="%s" fill-opacity="%.2f" stroke="none"/>`, data, paint, alpha*opacity)
		}
		if width == 0 {
			width = 3
		}
		filter := ""
		if id := d.filterFor(o.Effects); id != "" {
			filter = fmt.Sprintf(` filter="url(#%s)"`, id)
		}
		fmt.Fprintf(b, `<path d="%s" fill="none" stroke="%s" stroke-width="%.1f" stroke-opacity="%.2f" `+
			`stroke-linecap="round" stroke-linejoin="round"%s/>`, data, stroke, width, opacity, filter)
	case ObjLine, ObjArrow:
		marker := ""
		if o.Type == ObjArrow {
			marker = ` marker-end="url(#ah)"`
		}
		fmt.Fprintf(b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.1f" `+
			`stroke-opacity="%.2f" stroke-linecap="round"%s/>`, o.X, o.Y, o.X2, o.Y2, stroke, width, opacity, marker)
	case ObjRect:
		radius := 8.0
		if o.Radius != nil {
			radius = math.Max(0, *o.Radius)
		}
		fmt.Fprintf(b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="%.1f"%s/>`,
			o.X, o.Y, o.W, o.H, radius, d.paintAttrs(o, false))
	case ObjEllipse:
		fmt.Fprintf(b, `<ellipse cx="%.1f" cy="%.1f" rx="%.1f" ry="%.1f"%s/>`,
			o.X+o.W/2, o.Y+o.H/2, o.W/2, o.H/2, d.paintAttrs(o, false))
	case ObjDiamond:
		fmt.Fprintf(b, `<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f"%s/>`,
			o.X+o.W/2, o.Y, o.X+o.W, o.Y+o.H/2, o.X+o.W/2, o.Y+o.H, o.X, o.Y+o.H/2, d.paintAttrs(o, false))
	case ObjPolygon:
		fmt.Fprintf(b, `<polygon points="%s"%s/>`, pointsAttr(polygonPoints(o)), d.paintAttrs(o, false))
	case ObjSticky:
		note := sceneHex(o.Color)
		radius := 6.0
		if o.Radius != nil {
			radius = math.Max(0, *o.Radius)
		}
		fmt.Fprintf(b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="%.1f" fill="%s" fill-opacity="0.35" `+
			`stroke="%s" stroke-width="1"/>`, o.X, o.Y, o.W, o.H, radius, note, note)
		writeSVGText(b, o.Text, o.X+12, o.Y+26, 16, sceneHex("ink"))
	case ObjText:
		size := o.Size
		if size == 0 {
			size = 18
		}
		if o.PathRef != "" {
			if ref, ok := d.objectByID(o.PathRef); ok {
				d.writeTextOnPath(b, o, ref, size, stroke)
				return
			}
		}
		writeSVGText(b, o.Text, o.X, o.Y, size, stroke)
	case ObjComment:
		tint := sceneHex(o.Color)
		if o.Resolved {
			tint = sceneHex("green")
		}
		r := float64(commentPin) / 2
		alpha := 1.0
		if o.Resolved {
			alpha = 0.55
		}
		fmt.Fprintf(b, `<circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s" fill-opacity="%.2f"/>`,
			o.X+r, o.Y+r, r, tint, alpha)
		writeSVGText(b, strconv.Itoa(1+len(o.Replies)), o.X+r-4, o.Y+r+5, 14, sceneHex("chalk"))
	case ObjImage:
		if o.Src == "" || d.resolve == nil {
			return
		}
		mime, data, err := d.resolve(StorageKey(o.Src))
		if err != nil || len(data) == 0 {
			return
		}
		filter := ""
		if id := d.filterFor(o.Effects); id != "" {
			filter = fmt.Sprintf(` filter="url(#%s)"`, id)
		}
		clip := ""
		if o.Radius != nil && *o.Radius > 0 {
			id := d.id("cp")
			fmt.Fprintf(&d.defs, `<clipPath id="%s"><rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="%.1f"/></clipPath>`,
				id, o.X, o.Y, o.W, o.H, *o.Radius)
			clip = fmt.Sprintf(` clip-path="url(#%s)"`, id)
		}
		fmt.Fprintf(b, `<image x="%.1f" y="%.1f" width="%.1f" height="%.1f" opacity="%.2f"%s%s xlink:href="data:%s;base64,%s"/>`,
			o.X, o.Y, o.W, o.H, opacity, filter, clip, mime, base64Encode(data))
	}
}

func (d *svgDoc) objectByID(id string) (SceneObject, bool) {
	for _, o := range d.scene.Objects {
		if o.ID == id {
			return o, true
		}
	}
	return SceneObject{}, false
}

/*
writeTextOnPath — надпись вдоль контура. Опорный контур кладём в defs

	отдельным путём: <textPath> умеет ссылаться только на элемент по id.
*/
func (d *svgDoc) writeTextOnPath(b *strings.Builder, o SceneObject, ref SceneObject, size float64, color string) {
	data := shapePathData(ref)
	if data == "" {
		writeSVGText(b, o.Text, o.X, o.Y, size, color)
		return
	}
	id := d.id("tp")
	fmt.Fprintf(&d.defs, `<path id="%s" d="%s" fill="none"/>`, id, data)
	offset := math.Min(100, math.Max(0, o.PathOffset))
	fmt.Fprintf(b, `<text font-family="Inter, Arial, sans-serif" font-size="%.1f" fill="%s">`+
		`<textPath xlink:href="#%s" startOffset="%.1f%%">%s</textPath></text>`,
		size, color, id, offset, escapeXML(o.Text))
}

// fillAlpha — сила заливки фигуры: обычная полупрозрачна (за ней виден холст),
// «сплошная» приходит из заливки ведром и градиента.
func fillAlpha(o SceneObject) float64 {
	if o.Fill == "" {
		return 0
	}
	base := 0.35
	if o.Solid {
		base = 1
	}
	return base
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
