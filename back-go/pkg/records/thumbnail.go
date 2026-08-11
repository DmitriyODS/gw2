package records

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"

	// Регистрация декодеров: image.Decode узнаёт формат по сигнатуре файла.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

// ThumbMax — сторона миниатюры (по большему измерению). Хватает и таблице
// записей, и карточке на телефоне при удвоенной плотности экрана.
const ThumbMax = 320

// thumbQuality — качество JPEG миниатюры: на 320 px разница с 90 не видна, а
// вес втрое меньше.
const thumbQuality = 78

// Thumbnail — уменьшенная копия картинки для показа в таблице: без неё строка
// с фотографией на 3 МБ тянула бы весь оригинал, а на странице их тридцать.
//
// ok=false — миниатюра не нужна или невозможна: картинка и так меньше max,
// формат не декодируется (webp и прочая экзотика) либо данные битые. Вызывающий
// в этом случае показывает оригинал — превью не критично.
//
// Формат результата выбирается по содержимому: есть полупрозрачные пиксели —
// PNG (в JPEG прозрачность стала бы чёрным), иначе JPEG.
func Thumbnail(data []byte, max int) ([]byte, bool) {
	if max <= 0 {
		max = ThumbMax
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 || (w <= max && h <= max) {
		return nil, false
	}

	dstW, dstH := max, max
	if w > h {
		dstH = maxInt(1, h*max/w)
	} else {
		dstW = maxInt(1, w*max/h)
	}

	dst, opaque := downscale(src, dstW, dstH)

	var buf bytes.Buffer
	if opaque {
		if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: thumbQuality}); err != nil {
			return nil, false
		}
	} else if err := png.Encode(&buf, dst); err != nil {
		return nil, false
	}
	// Миниатюра тяжелее оригинала — значит оригинал уже оптимален (мелкий PNG,
	// прогрессивный JPEG); второй файл в хранилище тогда не нужен.
	if buf.Len() >= len(data) {
		return nil, false
	}
	return buf.Bytes(), opaque
}

// ThumbExt — расширение файла миниатюры по признаку непрозрачности (его
// возвращает Thumbnail вторым значением).
func ThumbExt(opaque bool) string {
	if opaque {
		return ".jpg"
	}
	return ".png"
}

// downscale — усреднение по прямоугольнику исходника на каждый пиксель приёмника
// (box filter). Для уменьшения этого достаточно: результат мягче и честнее, чем
// у выборки ближайшего соседа, а зависимостей не нужно.
func downscale(src image.Image, dstW, dstH int) (*image.NRGBA, bool) {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))
	opaque := true

	for y := 0; y < dstH; y++ {
		y0 := b.Min.Y + y*b.Dy()/dstH
		y1 := maxInt(y0+1, b.Min.Y+(y+1)*b.Dy()/dstH)
		for x := 0; x < dstW; x++ {
			x0 := b.Min.X + x*b.Dx()/dstW
			x1 := maxInt(x0+1, b.Min.X+(x+1)*b.Dx()/dstW)

			var sr, sg, sb, sa, n uint64
			for yy := y0; yy < y1; yy++ {
				for xx := x0; xx < x1; xx++ {
					// RGBA() отдаёт premultiplied 16-битные компоненты.
					r, g, bl, a := src.At(xx, yy).RGBA()
					sr += uint64(r)
					sg += uint64(g)
					sb += uint64(bl)
					sa += uint64(a)
					n++
				}
			}
			if n == 0 {
				continue
			}
			c := color.NRGBA64Model.Convert(color.RGBA64{
				R: uint16(sr / n), G: uint16(sg / n), B: uint16(sb / n), A: uint16(sa / n),
			}).(color.NRGBA64)
			if c.A < 0xffff {
				opaque = false
			}
			dst.SetNRGBA(x, y, color.NRGBA{
				R: uint8(c.R >> 8), G: uint8(c.G >> 8), B: uint8(c.B >> 8), A: uint8(c.A >> 8),
			})
		}
	}
	return dst, opaque
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
