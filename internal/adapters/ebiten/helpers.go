package ebitenui

import (
	"bytes"
	"image"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	_ "embed"
	_ "image/png" // must to PNG decoder
)

type win_sceneType int

const (
	Title win_sceneType = iota
	Normal
)

// COLORS
var ColorBlue = color.RGBA{27, 42, 74, 255}
var ColorBlack = color.RGBA{20, 20, 20, 255}

func drawRealisticText(dst *ebiten.Image, textStr string, x, y float64, face text.Face, baseColor color.RGBA) {
	// 1. Default Draw
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(baseColor)
	text.Draw(dst, textStr, face, op)

	// Get Alternative colors
	bleedColor1, bleedColor2 := getBleedColors(baseColor)

	// Move Text and darken and low opacity color
	op2 := &text.DrawOptions{}
	op2.GeoM.Translate(x+0.5, y+0.5)
	op2.ColorScale.ScaleWithColor(bleedColor1)
	text.Draw(dst, textStr, face, op2)

	// 3. Imperfections
	op3 := &text.DrawOptions{}
	op3.GeoM.Translate(x-0.3, y+0.3)
	op3.ColorScale.ScaleWithColor(bleedColor2)
	text.Draw(dst, textStr, face, op3)
}

func getBleedColors(base color.RGBA) (color.RGBA, color.RGBA) {
	// darken and low opacity color
	bleed1 := color.RGBA{
		R: uint8(float64(base.R) * 0.6),
		G: uint8(float64(base.G) * 0.6),
		B: uint8(float64(base.B) * 0.6),
		A: 30,
	}

	// imperfections
	bleed2 := color.RGBA{
		R: uint8(float64(base.R) * 0.4),
		G: uint8(float64(base.G) * 0.4),
		B: uint8(float64(base.B) * 0.4),
		A: 20,
	}

	return bleed1, bleed2
}

// MustLoadFont crete a text.Face from the bytes of a OpenType/TTF with the desired size.
func MustLoadFont(fontData []byte, size float64) text.Face {
	tt, err := opentype.Parse(fontData)
	if err != nil {
		log.Fatalf("Error crítico al parsear la fuente: %v", err)
	}

	baseFace, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatalf("Critico error creating the font Face (size %.0f): %v", size, err)
	}

	return text.NewGoXFace(baseFace)
}

// MustLoadEbitenImage byte converte to *ebiten.Image or stops if not exist.
func mustLoadEbitenImage(data []byte) *ebiten.Image {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Fatalf("Critical Error: the image can't be decoded: %v", err)
	}
	return ebiten.NewImageFromImage(img)
}
