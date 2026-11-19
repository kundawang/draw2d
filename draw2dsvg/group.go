// Copyright 2015 The draw2d Authors. All rights reserved.
// created: 20/11/2026 by draw2d contributors

package draw2dsvg

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/llgcode/draw2d"
)

// ErrNoOpenGroup is returned by EndGroup when no group is open.
var ErrNoOpenGroup = errors.New("draw2dsvg: EndGroup called without a matching BeginGroup")

// BeginGroup opens an isolated transparency group. Elements drawn until the
// matching EndGroup are written inside a nested <g opacity="..."> element, so
// a renderer first composites them together and then blends the group onto
// the parent canvas as a whole — overlapping translucent shapes do not
// darken each other's overlap. Groups nest; parent and child opacities
// multiply. The current transformation matrix keeps applying to the elements
// inside the group.
//
// An empty group is dropped from the document at EndGroup. A group left open
// when the document is saved stays in the document, so no content is lost.
func (gc *GraphicContext) BeginGroup(opacity float64) error {
	if err := draw2d.ValidateGroupOpacity(opacity); err != nil {
		return err
	}
	if len(gc.groupStack) >= draw2d.MaxGroupNestingDepth {
		return fmt.Errorf("draw2dsvg: transparency group nesting exceeds the maximum depth of %d", draw2d.MaxGroupNestingDepth)
	}
	group := &Group{Opacity: strconv.FormatFloat(opacity, 'f', -1, 64)}
	container := gc.currentContainer()
	*container = append(*container, group)
	gc.groupStack = append(gc.groupStack, group)
	return nil
}

// EndGroup closes the most recently opened group. If the group has no
// content it is removed from the document.
func (gc *GraphicContext) EndGroup() error {
	n := len(gc.groupStack)
	if n == 0 {
		return ErrNoOpenGroup
	}
	group := gc.groupStack[n-1]
	gc.groupStack = gc.groupStack[:n-1]
	if group.empty() {
		gc.removeFromContainer(group)
	}
	return nil
}

// GroupDepth returns the number of currently open transparency groups.
func (gc *GraphicContext) GroupDepth() int {
	return len(gc.groupStack)
}

// empty reports whether the group holds no drawable content.
func (g *Group) empty() bool {
	return len(g.Groups) == 0 && len(g.Paths) == 0 && len(g.Texts) == 0 && g.Image == nil
}

// removeFromContainer removes group from the group list it was appended to.
func (gc *GraphicContext) removeFromContainer(group *Group) {
	container := gc.currentContainer()
	groups := *container
	for i, g := range groups {
		if g == group {
			*container = append(groups[:i], groups[i+1:]...)
			return
		}
	}
}
