package main

import (
	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/goregular"
)

func main() {
	dc := gg.NewContext(400, 400)

	dc.SetRGB(1, 1, 1)
	dc.Clear()

	dc.SetRGB(0, 0, 0)

	drawHeader(dc)
	drawText(dc)

	dc.SavePNG("out.png")
}

func drawHeader(dc *gg.Context) {
	font, _ := truetype.Parse(goregular.TTF)
	face := truetype.NewFace(font, &truetype.Options{
		Size: 40,
	})

	dc.SetFontFace(face)
	dc.DrawString("Card Header", 20, 50)
	dc.SetLineWidth(1)
	dc.SetRGB(0, 0, 0)
	dc.DrawLine(20, 54, 380, 54)
	dc.Stroke()
}

func drawText(dc *gg.Context) {
	font, _ := truetype.Parse(goregular.TTF)
	face := truetype.NewFace(font, &truetype.Options{
		Size: 22,
	})

	text := "Lorem Ipsum is simply dummy text of the printing and typesetting industry. Lorem Ipsum has been the industry's standard dummy text ever since the 1500s, when an unknown printer took a galley of type and scrambled it to make a type specimen book."

	dc.SetFontFace(face)
	dc.DrawStringWrapped(text, 20, 70, 0, 0, 370, 1, gg.AlignLeft)
}
