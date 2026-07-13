package service

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"

	xdraw "golang.org/x/image/draw"
)

// thumbMaxDim — макс. сторона превью-картинки чата (px): хватает для ретины при
// типичной ширине пузыря, но по трафику в разы легче оригинала.
const thumbMaxDim = 600

// makeThumbnail уменьшает картинку так, чтобы большая сторона не превышала
// thumbMaxDim, сплющивает прозрачность на белый фон и кодирует в JPEG.
// ok=false — данные не декодировались как растровая картинка (svg/webp/битый
// файл) ЛИБО превью не легче оригинала: тогда превью не делаем, клиент покажет
// исходник.
func makeThumbnail(data []byte) ([]byte, bool) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil, false
	}
	nw, nh := w, h
	if w > thumbMaxDim || h > thumbMaxDim {
		if w >= h {
			nw, nh = thumbMaxDim, h*thumbMaxDim/w
		} else {
			nw, nh = w*thumbMaxDim/h, thumbMaxDim
		}
	}
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	// Белый фон под возможную прозрачность — JPEG альфу не хранит.
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 78}); err != nil {
		return nil, false
	}
	// Превью тяжелее оригинала (уже сжатый мелкий JPEG) — смысла нет.
	if buf.Len() >= len(data) {
		return nil, false
	}
	return buf.Bytes(), true
}
