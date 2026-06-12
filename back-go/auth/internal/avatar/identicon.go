// Package avatar — identicon и файлы аватарок (общий uploads-каталог).
package avatar

import (
	"bytes"
	"crypto/sha256"
	"image"
	"image/color"
	"image/png"
	"strconv"
)

const identiconSize = 192

// Identicon — pixel-art 8×8 PNG. Алгоритм байт-в-байт повторяет прежний
// back/app/utils/avatar.py: sha256 от строкового id, оттенок из первого
// байта, 4 уникальных столбца + зеркало, масштабирование без сглаживания.
func Identicon(userID int64) []byte {
	data := sha256.Sum256([]byte(strconv.FormatInt(userID, 10)))

	hue := float64(data[0]) / 255.0
	fg := hslToRGB(hue, 0.70, 0.50)
	fg2 := hslToRGB(hue, 0.70, 0.35)
	bg := color.RGBA{248, 248, 252, 255}

	const grid, half = 8, 4
	cells := [grid][grid]color.RGBA{}
	for row := 0; row < grid; row++ {
		for col := 0; col < grid; col++ {
			cells[row][col] = bg
		}
	}
	for row := 0; row < grid; row++ {
		for col := 0; col < half; col++ {
			b := data[(row*half+col)%len(data)]
			if b > 100 {
				c := fg
				if b > 200 {
					c = fg2
				}
				cells[row][col] = c
				cells[row][grid-1-col] = c
			}
		}
	}

	img := image.NewRGBA(image.Rect(0, 0, identiconSize, identiconSize))
	scale := identiconSize / grid
	for y := 0; y < identiconSize; y++ {
		for x := 0; x < identiconSize; x++ {
			img.SetRGBA(x, y, cells[y/scale][x/scale])
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// hslToRGB — преобразование как в colorsys.hls_to_rgb (Python).
func hslToRGB(h, s, l float64) color.RGBA {
	var m2 float64
	if l <= 0.5 {
		m2 = l * (1 + s)
	} else {
		m2 = l + s - l*s
	}
	m1 := 2*l - m2
	r := hueToRGB(m1, m2, h+1.0/3.0)
	g := hueToRGB(m1, m2, h)
	b := hueToRGB(m1, m2, h-1.0/3.0)
	return color.RGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), 255}
}

func hueToRGB(m1, m2, h float64) float64 {
	if h < 0 {
		h++
	}
	if h > 1 {
		h--
	}
	switch {
	case h < 1.0/6.0:
		return m1 + (m2-m1)*6*h
	case h < 0.5:
		return m2
	case h < 2.0/3.0:
		return m1 + (m2-m1)*(2.0/3.0-h)*6
	default:
		return m1
	}
}
