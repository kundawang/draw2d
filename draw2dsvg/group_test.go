// Copyright 2015 The draw2d Authors. All rights reserved.
// created: 20/11/2026 by draw2d contributors

package draw2dsvg

import (
	"encoding/xml"
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/llgcode/draw2d"
)

var _ draw2d.GroupContext = (*GraphicContext)(nil)

func fillRectPath(gc *GraphicContext, x1, y1, x2, y2 float64, c color.Color) {
	gc.SetFillColor(c)
	gc.MoveTo(x1, y1)
	gc.LineTo(x2, y1)
	gc.LineTo(x2, y2)
	gc.LineTo(x1, y2)
	gc.Close()
	gc.Fill()
}

// Two overlapping translucent rectangles plus a nested group must be written
// as nested <g opacity="..."> elements so a renderer composites each group
// before blending it onto its parent.
func TestGroupXml(t *testing.T) {
	svg := NewSvg()
	gc := NewGraphicContext(svg)

	if err := gc.BeginGroup(0.5); err != nil {
		t.Fatal(err)
	}
	// color.RGBA values are written to the svg as-is (the svg backend does
	// not composite); NRGBA would be emitted with premultiplied channels.
	fillRectPath(gc, 10, 10, 50, 50, color.RGBA{255, 0, 0, 128})
	fillRectPath(gc, 30, 30, 70, 70, color.RGBA{0, 0, 255, 128})
	if err := gc.BeginGroup(0.25); err != nil {
		t.Fatal(err)
	}
	fillRectPath(gc, 40, 40, 60, 60, color.NRGBA{0, 128, 0, 255})
	if err := gc.EndGroup(); err != nil {
		t.Fatal(err)
	}
	if err := gc.EndGroup(); err != nil {
		t.Fatal(err)
	}
	if gc.GroupDepth() != 0 {
		t.Fatalf("GroupDepth = %d, want 0", gc.GroupDepth())
	}

	expectedOut := `<svg xmlns="http://www.w3.org/2000/svg" fill="none" stroke="none">
  <defs></defs>
  <g opacity="0.5">
    <g fill="rgba(255,0,0,0.502)" fill-rule="evenodd">
      <path d="M 10,10 L 50,10 L 50,50 L 10,50 Z"></path>
    </g>
    <g fill="rgba(0,0,255,0.502)" fill-rule="evenodd">
      <path d="M 30,30 L 70,30 L 70,70 L 30,70 Z"></path>
    </g>
    <g opacity="0.25">
      <g fill="#008000" fill-rule="evenodd">
        <path d="M 40,40 L 60,40 L 60,60 L 40,60 Z"></path>
      </g>
    </g>
  </g>
</svg>`

	out, err := xml.MarshalIndent(svg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != expectedOut {
		t.Errorf("svg output is not as expected\ngot:\n%s\n\nwant:\n%s\n", out, expectedOut)
	}
}

// An empty group is dropped from the document at EndGroup.
func TestGroupEmptyDropped(t *testing.T) {
	svg := NewSvg()
	gc := NewGraphicContext(svg)
	if err := gc.BeginGroup(0.5); err != nil {
		t.Fatal(err)
	}
	if err := gc.EndGroup(); err != nil {
		t.Fatal(err)
	}
	out, err := xml.Marshal(svg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "<g") {
		t.Errorf("empty group should be dropped, got:\n%s", out)
	}
}

// A group left open when the document is saved stays in the document, so no
// content is lost.
func TestGroupUnclosedKept(t *testing.T) {
	svg := NewSvg()
	gc := NewGraphicContext(svg)
	if err := gc.BeginGroup(0.5); err != nil {
		t.Fatal(err)
	}
	fillRectPath(gc, 0, 0, 10, 10, color.NRGBA{255, 0, 0, 255})
	// no EndGroup
	out, err := xml.Marshal(svg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `opacity="0.5"`) || !strings.Contains(string(out), "<path") {
		t.Errorf("unclosed group content should be kept, got:\n%s", out)
	}
}

func TestGroupInvalidOpacity(t *testing.T) {
	for _, opacity := range []float64{-0.1, 1.1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		svg := NewSvg()
		gc := NewGraphicContext(svg)
		if err := gc.BeginGroup(opacity); err == nil {
			t.Errorf("BeginGroup(%v) should fail", opacity)
		}
		if gc.GroupDepth() != 0 {
			t.Errorf("BeginGroup(%v) failed but left a group open", opacity)
		}
	}
}

func TestGroupEndWithoutBegin(t *testing.T) {
	gc := NewGraphicContext(NewSvg())
	if err := gc.EndGroup(); err != ErrNoOpenGroup {
		t.Errorf("EndGroup without BeginGroup = %v, want ErrNoOpenGroup", err)
	}
}

func TestGroupNestingDepthLimit(t *testing.T) {
	gc := NewGraphicContext(NewSvg())
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
}
