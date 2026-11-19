// Copyright 2010 The draw2d Authors. All rights reserved.
// created: 20/11/2026 by draw2d contributors

package draw2dimg

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/golang/freetype/raster"
	"github.com/llgcode/draw2d"
)

// MaxGroupLayerPixels bounds the number of pixels of the offscreen layer
// allocated by BeginGroup. The layer covers the current canvas bounds, so a
// group on a canvas larger than this limit is rejected with an error instead
// of allocating an unbounded amount of memory. It is a variable so
// applications (and tests) can tune it.
var MaxGroupLayerPixels = 1 << 26 // 64 Mi pixels, 256 MiB per RGBA layer

// ErrNoOpenGroup is returned by EndGroup when no group is open.
var ErrNoOpenGroup = errors.New("draw2dimg: EndGroup called without a matching BeginGroup")

// groupLayer is one open transparency group: the parent drawing target the
// group will be composited back onto, and the group opacity.
type groupLayer struct {
	target  draw.Image
	painter Painter
	opacity float64
}

// BeginGroup opens an isolated transparency group. Subsequent drawing goes to
// an offscreen layer covering the current canvas bounds; EndGroup blends that
// layer back onto the parent canvas as a whole with the given opacity, so
// overlapping translucent shapes inside the group do not darken each other's
// overlap. Groups nest; parent and child opacities multiply. The current
// transformation matrix applies to drawing inside the group as usual, and the
// composited result is clipped to the parent canvas bounds.
//
// An empty group composites a fully transparent layer and is a no-op. A group
// left open is never composited: its content stays in the offscreen layer and
// does not appear on the parent canvas.
func (gc *GraphicContext) BeginGroup(opacity float64) error {
	if err := draw2d.ValidateGroupOpacity(opacity); err != nil {
		return err
	}
	if len(gc.groups) >= draw2d.MaxGroupNestingDepth {
		return fmt.Errorf("draw2dimg: transparency group nesting exceeds the maximum depth of %d", draw2d.MaxGroupNestingDepth)
	}
	bounds := gc.img.Bounds()
	if pixels := bounds.Dx() * bounds.Dy(); pixels > MaxGroupLayerPixels {
		return fmt.Errorf("draw2dimg: group layer of %d pixels exceeds the limit of %d pixels", pixels, MaxGroupLayerPixels)
	}
	layer := image.NewRGBA(bounds)
	gc.groups = append(gc.groups, groupLayer{target: gc.img, painter: gc.painter, opacity: opacity})
	gc.img = layer
	gc.painter = raster.NewRGBAPainter(layer)
	return nil
}

// EndGroup closes the most recently opened group and composites its offscreen
// layer onto the parent canvas with the group opacity.
func (gc *GraphicContext) EndGroup() error {
	n := len(gc.groups)
	if n == 0 {
		return ErrNoOpenGroup
	}
	top := gc.groups[n-1]
	gc.groups = gc.groups[:n-1]
	layer := gc.img
	gc.img = top.target
	gc.painter = top.painter
	// Blend the whole layer at once: the uniform mask scales both color and
	// alpha by the group opacity (source-over with constant alpha).
	mask := image.NewUniform(color.Alpha{A: uint8(top.opacity*255 + 0.5)})
	draw.DrawMask(gc.img, layer.Bounds(), layer, layer.Bounds().Min, mask, image.Point{}, draw.Over)
	return nil
}

// GroupDepth returns the number of currently open transparency groups.
func (gc *GraphicContext) GroupDepth() int {
	return len(gc.groups)
}
