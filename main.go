package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"os"

	"github.com/nfnt/resize"
)

type AtFunc func(x, y int) color.Color

type AlphaMask struct {
	Outer, Inner image.Rectangle
	Reverse      bool
	AtFunc       func(m *AlphaMask) AtFunc
}

func (m *AlphaMask) ColorModel() color.Model {
	return color.AlphaModel
}

func (m *AlphaMask) Bounds() image.Rectangle {
	return m.Outer
}

func (m *AlphaMask) At(x, y int) color.Color {
	return m.AtFunc(m)(x, y)
}

func alphaOf(inner, reverse bool) color.Color {
	if inner {
		if reverse {
			return color.Alpha{A: 0}
		}
		return color.Alpha{A: 255}
	}
	if reverse {
		return color.Alpha{A: 255}
	}
	return color.Alpha{A: 0}
}

func circleAtFunc(m *AlphaMask) AtFunc {
	return func(x, y int) color.Color {
		x -= m.Inner.Min.X
		y -= m.Inner.Min.Y
		var X, Y, R = m.Inner.Dx() / 2, m.Inner.Dy() / 2, m.Inner.Dx() / 2
		xx, yy, rr := float64(x-X)+0.5, float64(y-Y)+0.5, float64(R)
		return alphaOf(xx*xx+yy*yy < rr*rr, m.Reverse)
	}
}

func rectangleAtFunc(m *AlphaMask) AtFunc {
	return func(x, y int) color.Color {
		return alphaOf(image.Pt(x, y).In(m.Inner), m.Reverse)
	}
}

var (
	usage = `Usage:
	slack-icon-merger {SRC_IMAGE_FILE} {OVER_IMAGE_FILE}
`
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	var (
		srcImgPath  = os.Args[1]
		overImgPath = os.Args[2]
	)

	srcImgFile, err := os.Open(srcImgPath)
	if err != nil {
		panic(err)
	}
	defer srcImgFile.Close()
	overImgFile, err := os.Open(overImgPath)
	if err != nil {
		panic(err)
	}
	defer overImgFile.Close()

	srcImg, _, err := image.Decode(srcImgFile)
	if err != nil {
		panic(err)
	}
	overImg, _, err := image.Decode(overImgFile)
	if err != nil {
		panic(err)
	}

	var (
		w, h = int(float32(srcImg.Bounds().Dx()) / 3), int(float32(srcImg.Bounds().Dy()) / 3)
		bw   = 1
	)

	circleRect := image.Rectangle{
		Min: image.Pt(srcImg.Bounds().Size().X-w-2*bw, 0),
		Max: image.Pt(srcImg.Bounds().Size().X, h+2*bw),
	}
	cornerRect := image.Rectangle{
		Min: image.Pt(srcImg.Bounds().Size().X-w/2-bw, 0),
		Max: image.Pt(srcImg.Bounds().Size().X, h/2+bw),
	}
	dogRect := image.Rectangle{
		Min: image.Pt(srcImg.Bounds().Size().X-w-bw, bw),
		Max: image.Pt(srcImg.Bounds().Size().X-bw, int(h)+bw),
	}

	outRect := image.Rectangle{Min: image.ZP, Max: srcImg.Bounds().Size()}
	out := image.NewRGBA(outRect)
	draw.DrawMask(out, outRect, srcImg, image.ZP,
		&AlphaMask{outRect, circleRect, true, circleAtFunc}, image.ZP,
		draw.Src)
	draw.DrawMask(out, outRect, out, image.ZP,
		&AlphaMask{outRect, cornerRect, true, rectangleAtFunc}, image.ZP,
		draw.Src)

	overImg = resize.Resize(uint(w), uint(h), overImg, resize.NearestNeighbor)
	mask := &AlphaMask{
		image.Rectangle{Min: image.ZP, Max: dogRect.Size()},
		image.Rectangle{Min: image.ZP, Max: dogRect.Size()},
		false,
		circleAtFunc,
	}
	draw.DrawMask(out, dogRect, overImg, image.ZP, mask, image.ZP, draw.Over)

	outPath := "./merged.png"
	outFile, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}
	defer outFile.Close()

	err = png.Encode(outFile, out)
	if err != nil {
		panic(err)
	}

	fmt.Printf("PNG image generated: %s\n", outPath)
}
