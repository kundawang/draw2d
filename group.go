// Copyright 2010 The draw2d Authors. All rights reserved.
// created: 20/11/2026 by draw2d contributors

package draw2d

import (
	"errors"
	"math"
)

// MaxGroupNestingDepth is the maximum number of transparency groups that may
// be open at the same time on a graphic context. BeginGroup returns an error
// once this depth is reached. The limit keeps pathological nesting from
// exhausting memory (each image-backend group allocates an offscreen layer).
const MaxGroupNestingDepth = 64

// ErrGroupOpacity is returned by BeginGroup when the requested group opacity
// is NaN, infinite, or outside the range [0, 1].
var ErrGroupOpacity = errors.New("draw2d: group opacity must be a finite number in [0, 1]")

// GroupContext is implemented by graphic contexts that support isolated
// transparency groups (draw2dimg, draw2dsvg).
//
// Drawing commands issued between BeginGroup and EndGroup are composed
// together into a single layer first; the layer is then blended onto the
// parent canvas as a whole with the group opacity. This avoids darkening
// the overlap of two translucent shapes the way per-shape alpha does.
// Groups nest: a child group is composited into its parent group's layer,
// so parent and child opacities multiply. The current transformation
// matrix and clipping of the parent apply to the composited result.
//
// Groups are independent of Save/Restore. A group left open when drawing
// finishes is backend specific: the image backend never composites it
// (its content is discarded with the layer), the svg backend keeps the
// open group in the document so no content is lost.
type GroupContext interface {
	GraphicContext
	// BeginGroup opens an isolated transparency group with the given
	// opacity in [0, 1]. It returns ErrGroupOpacity for invalid opacity
	// and an error when MaxGroupNestingDepth is exceeded.
	BeginGroup(opacity float64) error
	// EndGroup closes the most recent BeginGroup and composites the
	// group onto its parent. It returns an error when no group is open.
	EndGroup() error
}

// ValidateGroupOpacity reports whether opacity is usable as a group opacity,
// i.e. a finite number in [0, 1]. It returns ErrGroupOpacity otherwise.
func ValidateGroupOpacity(opacity float64) error {
	if math.IsNaN(opacity) || math.IsInf(opacity, 0) || opacity < 0 || opacity > 1 {
		return ErrGroupOpacity
	}
	return nil
}
