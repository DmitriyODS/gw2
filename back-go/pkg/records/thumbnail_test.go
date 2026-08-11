package records

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}

// Пёстрая картинка: одноцветную PNG сжимает так, что миниатюра оказывается
// тяжелее оригинала и Thumbnail законно отказывается её делать.
func noisy(w, h int, alpha uint8) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8((x * 7) % 256), G: uint8((y * 11) % 256), B: uint8((x * y) % 256), A: alpha,
			})
		}
	}
	return img
}

func TestThumbnailDownscalesLargeImage(t *testing.T) {
	data := encodePNG(t, noisy(1000, 500, 255))

	thumb, opaque := Thumbnail(data, 200)
	if thumb == nil {
		t.Fatal("миниатюра не построена")
	}
	if !opaque {
		t.Fatal("картинка без прозрачности должна уходить в JPEG")
	}
	if len(thumb) >= len(data) {
		t.Fatalf("миниатюра не легче оригинала: %d ≥ %d", len(thumb), len(data))
	}

	img, err := jpeg.Decode(bytes.NewReader(thumb))
	if err != nil {
		t.Fatalf("миниатюра не читается как JPEG: %v", err)
	}
	if got := img.Bounds().Dx(); got != 200 {
		t.Fatalf("ширина миниатюры %d, ожидалось 200", got)
	}
	if got := img.Bounds().Dy(); got != 100 {
		t.Fatalf("высота миниатюры %d, пропорции не сохранены", got)
	}
}

func TestThumbnailKeepsAlphaAsPNG(t *testing.T) {
	data := encodePNG(t, noisy(800, 800, 128))

	thumb, opaque := Thumbnail(data, 100)
	if thumb == nil {
		t.Fatal("миниатюра не построена")
	}
	if opaque {
		t.Fatal("полупрозрачная картинка должна оставаться PNG")
	}
	if _, err := png.Decode(bytes.NewReader(thumb)); err != nil {
		t.Fatalf("миниатюра не читается как PNG: %v", err)
	}
	if ThumbExt(opaque) != ".png" {
		t.Fatalf("расширение %q, ожидалось .png", ThumbExt(opaque))
	}
}

func TestThumbnailSkipsSmallAndBrokenInput(t *testing.T) {
	small := encodePNG(t, noisy(120, 90, 255))
	if thumb, _ := Thumbnail(small, 320); thumb != nil {
		t.Fatal("картинке меньше порога миниатюра не нужна")
	}
	if thumb, _ := Thumbnail([]byte("не картинка"), 320); thumb != nil {
		t.Fatal("нечитаемые данные не должны давать миниатюру")
	}
}
