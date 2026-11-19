// Copyright 2010 The draw2d Authors. All rights reserved.
// created: 20/11/2026 by draw2d contributors

package draw2dimg

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"testing"

	"github.com/llgcode/draw2d"
)

var _ draw2d.GroupContext = (*GraphicContext)(nil)

// fillRect fills the axis-aligned rectangle (x1,y1)-(x2,y2) with c.
func fillRect(gc *GraphicContext, x1, y1, x2, y2 float64, c color.Color) {
	gc.SetFillColor(c)
	gc.MoveTo(x1, y1)
	gc.LineTo(x2, y1)
	gc.LineTo(x2, y2)
	gc.LineTo(x1, y2)
	gc.Close()
	gc.Fill()
}

func whiteCanvas(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)
	return img
}

func assertPixel(t *testing.T, img *image.RGBA, x, y int, want color.RGBA) {
	t.Helper()
	got := img.RGBAAt(x, y)
	const tol = 2 // rounding tolerance of the 16-bit rasterizer math
	if diff(got.R, want.R) > tol || diff(got.G, want.G) > tol || diff(got.B, want.B) > tol || diff(got.A, want.A) > tol {
		t.Errorf("pixel (%d,%d) = %v, want %v (±%d)", x, y, got, want, tol)
	}
}

func diff(a, b uint8) int {
	d := int(a) - int(b)
	if d < 0 {
		return -d
	}
	return d
}

// Two overlapping translucent rectangles inside a group must be composed
// together first, then faded as a whole: the overlap must not be darkened
// twice the way sequential per-shape alpha does.
func TestGroupOverlappingRects(t *testing.T) {
	// NRGBA: the freetype painter expects colors honoring the premultiplied
	// contract; NRGBA.RGBA() does the premultiplication.
	halfRed := color.NRGBA{255, 0, 0, 128}
	halfBlue := color.NRGBA{0, 0, 255, 128}

	// Grouped rendering: group opacity 0.5 over a white canvas.
	grouped := whiteCanvas(3, 1)
	gc := NewGraphicContext(grouped)
	if err := gc.BeginGroup(0.5); err != nil {
		t.Fatal(err)
	}
	fillRect(gc, 0, 0, 2, 1, halfRed)
	fillRect(gc, 1, 0, 3, 1, halfBlue)
	if err := gc.EndGroup(); err != nil {
		t.Fatal(err)
	}

	// Expected values, computed with straight-alpha compositing:
	//   layer A-only:  red a=0.5   -> x0.5 over white: (255,191,191)
	//   layer overlap: blue over red, a=0.75, rgb=(1/3,0,2/3)
	//                  -> x0.5 over white:            (191,159,223)
	//   layer B-only:  blue a=0.5  -> x0.5 over white: (191,191,255)
	assertPixel(t, grouped, 0, 0, color.RGBA{255, 191, 191, 255})
	assertPixel(t, grouped, 1, 0, color.RGBA{191, 159, 223, 255})
	assertPixel(t, grouped, 2, 0, color.RGBA{191, 191, 255, 255})

	// The same pixels must survive a PNG encode/decode round trip.
	var buf bytes.Buffer
	if err := png.Encode(&buf, grouped); err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	for x, want := range []color.RGBA{{255, 191, 191, 255}, {191, 159, 223, 255}, {191, 191, 255, 255}} {
		r, g, b, a := decoded.At(x, 0).RGBA()
		got := color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
		if diff(got.R, want.R) > 2 || diff(got.G, want.G) > 2 || diff(got.B, want.B) > 2 || diff(got.A, want.A) > 2 {
			t.Errorf("decoded png pixel (%d,0) = %v, want %v (±2)", x, got, want)
		}
	}

	// Sequential rendering of the same shapes without a group darkens the
	// overlap; the grouped result must differ from it there.
	sequential := whiteCanvas(3, 1)
	gc2 := NewGraphicContext(sequential)
	fillRect(gc2, 0, 0, 2, 1, halfRed)
	fillRect(gc2, 1, 0, 3, 1, halfBlue)

	g, s := grouped.RGBAAt(1, 0), sequential.RGBAAt(1, 0)
	if diff(g.R, s.R) < 50 || diff(g.G, s.G) < 50 {
		t.Errorf("grouped overlap %v too close to sequential overlap %v: overlap was darkened", g, s)
	}
	// Outside the overlap the sequential render keeps the shapes at their own
	// alpha (red x0.5 over white), while the group fades them once more by the
	// group opacity.
	assertPixel(t, sequential, 0, 0, color.RGBA{255, 127, 127, 255})
	assertPixel(t, sequential, 2, 0, color.RGBA{127, 127, 255, 255})
}

// Nested groups multiply opacities, and the current transform composes with
// the parent transform inside and outside the group.
func TestGroupNestedOpacityAndTransform(t *testing.T) {
	img := whiteCanvas(4, 1)
	gc := NewGraphicContext(img)

	gc.Translate(1, 0)
	if err := gc.BeginGroup(0.5); err != nil {
		t.Fatal(err)
	}
	gc.Translate(1, 0)
	if err := gc.BeginGroup(0.5); err != nil {
		t.Fatal(err)
	}
	fillRect(gc, 0, 0, 1, 1, color.RGBA{0, 0, 0, 255})
	if err := gc.EndGroup(); err != nil {
		t.Fatal(err)
	}
	if err := gc.EndGroup(); err != nil {
		t.Fatal(err)
	}

	// Effective alpha (0.5*0.5) of black over white -> gray ~191,
	// translated by 1+1 to pixel x=2.
	assertPixel(t, img, 0, 0, color.RGBA{255, 255, 255, 255})
	assertPixel(t, img, 1, 0, color.RGBA{255, 255, 255, 255})
	assertPixel(t, img, 2, 0, color.RGBA{191, 191, 191, 255})
	assertPixel(t, img, 3, 0, color.RGBA{255, 255, 255, 255})
}

// Drawing outside the canvas inside a group is clipped to the canvas bounds.
func TestGroupClippedToCanvas(t *testing.T) {
	img := whiteCanvas(3, 1)
	gc := NewGraphicContext(img)
	if err := gc.BeginGroup(1); err != nil {
		t.Fatal(err)
	}
	fillRect(gc, -10, -10, 20, 20, color.RGBA{0, 0, 0, 255})
	if err := gc.EndGroup(); err != nil {
		t.Fatal(err)
	}
	for x := 0; x < 3; x++ {
		assertPixel(t, img, x, 0, color.RGBA{0, 0, 0, 255})
	}
}

func TestGroupInvalidOpacity(t *testing.T) {
	for _, opacity := range []float64{-0.1, 1.1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		gc := NewGraphicContext(whiteCanvas(1, 1))
		if err := gc.BeginGroup(opacity); err == nil {
			t.Errorf("BeginGroup(%v) should fail", opacity)
		}
		if gc.GroupDepth() != 0 {
			t.Errorf("BeginGroup(%v) failed but left a group open", opacity)
		}
	}
}

func TestGroupEndWithoutBegin(t *testing.T) {
	gc := NewGraphicContext(whiteCanvas(1, 1))
	if err := gc.EndGroup(); err != ErrNoOpenGroup {
		t.Errorf("EndGroup without BeginGroup = %v, want ErrNoOpenGroup", err)
	}
}

func TestGroupNestingDepthLimit(t *testing.T) {
	gc := NewGraphicContext(whiteCanvas(1, 1))
	for i := 0; i < draw2d.MaxGroupNestingDepth; i++ {
		if err := gc.BeginGroup(1); err != nil {
			t.Fatalf("BeginGroup #%d failed: %v", i+1, err)
		}
	}
	if err := gc.BeginGroup(1); err == nil {
		t.Error("BeginGroup beyond MaxGroupNestingDepth should fail")
	}
	for i := 0; i < draw2d.MaxGroupNestingDepth; i++ {
		if err := gc.EndGroup(); err != nil {
			t.Fatalf("EndGroup #%d failed: %v", i+1, err)
		}
	}
	if gc.GroupDepth() != 0 {
		t.Error("all groups closed but GroupDepth != 0")
	}
}

func TestGroupLayerTooLarge(t *testing.T) {
	defer func(limit int) { MaxGroupLayerPixels = limit }(MaxGroupLayerPixels)
	MaxGroupLayerPixels = 10

	gc := NewGraphicContext(whiteCanvas(4, 4)) // 16 pixels > 10
	if err := gc.BeginGroup(1); err == nil {
		t.Error("BeginGroup on a canvas larger than MaxGroupLayerPixels should fail")
	}
	if gc.GroupDepth() != 0 {
		t.Error("rejected BeginGroup left a group open")
	}
}

// An empty group composites a transparent layer: the canvas is untouched.
func TestGroupEmptyIsNoOp(t *testing.T) {
	img := whiteCanvas(2, 1)
	gc := NewGraphicContext(img)
	if err := gc.BeginGroup(0.5); err != nil {
		t.Fatal(err)
	}
	if err := gc.EndGroup(); err != nil {
		t.Fatal(err)
	}
	assertPixel(t, img, 0, 0, color.RGBA{255, 255, 255, 255})
	assertPixel(t, img, 1, 0, color.RGBA{255, 255, 255, 255})
}

// A group that is never closed is never composited: its content does not
// leak onto the parent canvas.
func TestGroupUnclosedIsNotComposited(t *testing.T) {
	img := whiteCanvas(1, 1)
	gc := NewGraphicContext(img)
	if err := gc.BeginGroup(1); err != nil {
		t.Fatal(err)
	}
	fillRect(gc, 0, 0, 1, 1, color.RGBA{0, 0, 0, 255})
	if gc.GroupDepth() != 1 {
		t.Fatalf("GroupDepth = %d, want 1", gc.GroupDepth())
	}
	assertPixel(t, img, 0, 0, color.RGBA{255, 255, 255, 255})
	// Closing it later still composites correctly.
	if err := gc.EndGroup(); err != nil {
		t.Fatal(err)
	}
	assertPixel(t, img, 0, 0, color.RGBA{0, 0, 0, 255})
}
