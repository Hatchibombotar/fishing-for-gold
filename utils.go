package main

import (
	"image"
	"image/color"
	_ "image/png"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"embed"
)

var (
	whiteImage = ebiten.NewImage(3, 3)

	// whiteSubImage is an internal sub image of whiteImage.
	// Use whiteSubImage at DrawTriangles instead of whiteImage in order to avoid bleeding edges.
	whiteSubImage = whiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)

	whiteImageInitied = false
)

//go:embed assets/**
var emb embed.FS

func LoadImageFromPath(path string) *ebiten.Image {
	file, err := emb.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		panic(err)
	}

	sheet := ebiten.NewImageFromImage(img)

	return sheet
}

type Vector2 struct {
	x, y float64
}

func (v Vector2) Magnitude() float64 {
	return math.Sqrt(math.Pow(v.x, 2) + math.Pow(v.y, 2))
}
func (v Vector2) Unit() Vector2 {
	magnitude := v.Magnitude()
	return Vector2{v.x / magnitude, v.y / magnitude}
}
func (v Vector2) MultiplyByScalar(s float64) Vector2 {
	return Vector2{v.x * s, v.y * s}
}
func (v Vector2) Add(v2 Vector2) Vector2 {
	return Vector2{v.x + v2.x, v.y + v2.y}
}
func (v Vector2) UnpackInt() (int, int) {
	return int(v.x), int(v.y)
}
func (v Vector2) Unpack32() (float32, float32) {
	return float32(v.x), float32(v.y)
}
func (v Vector2) Unpack64() (float64, float64) {
	return v.x, v.y
}

func initWhiteImages() {
	whiteImage.Fill(color.White)
	whiteImageInitied = true
}

func StrokePath(screen *ebiten.Image, path *vector.Path, colour color.RGBA, width float32, x float32, y float32) {
	if !whiteImageInitied {
		initWhiteImages()
	}
	op_s := &vector.StrokeOptions{}
	op_s.Width = width
	op_s.LineJoin = vector.LineJoinRound
	vs, is := path.AppendVerticesAndIndicesForStroke(nil, nil, op_s)

	for i := range vs {
		vs[i].DstX = (vs[i].DstX + x)
		vs[i].DstY = (vs[i].DstY + y)
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = float32(colour.R) / float32(0xff)
		vs[i].ColorG = float32(colour.G) / float32(0xff)
		vs[i].ColorB = float32(colour.B) / float32(0xff)
		vs[i].ColorA = float32(colour.A) / float32(0xff)
	}

	op := &ebiten.DrawTrianglesOptions{}
	op.AntiAlias = false

	screen.DrawTriangles(vs, is, whiteSubImage, op)
}

func PointInRect(p Vector2, minPoint Vector2, maxPoint Vector2) bool {
	return p.x >= minPoint.x && p.x <= maxPoint.x &&
		p.y >= minPoint.y && p.y <= maxPoint.y
}
