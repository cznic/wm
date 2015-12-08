// Copyright 2015 The WM Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wm

import (
	"github.com/cznic/interval"
	"github.com/cznic/mathutil"
)

// Position represents 2D coordinates.
type Position struct {
	X, Y int
}

// In returns whether p is inside r.
func (p Position) In(r Rectangle) bool { return r.Has(p) }

// Rectangle represents a 2D area.
type Rectangle struct {
	Position
	Size
}

// NewRectangle returns a Rectangle from 4 coordinates.
func NewRectangle(x1, y1, x2, y2 int) Rectangle {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	return Rectangle{Position{x1, y1}, Size{x2 - x1 + 1, y2 - y1 + 1}}
}

// Clip sets r to the intersection of r and s and returns a boolean value indicating
// whether the result is of non zero size.
func (r *Rectangle) Clip(s Rectangle) bool {
	a := interval.Int{Cls: interval.LeftClosed, A: r.X, B: r.X + r.Width}
	b := interval.Int{Cls: interval.LeftClosed, A: s.X, B: s.X + s.Width}
	h0 := interval.Intersection(&a, &b)
	if h0.Class() == interval.Empty {
		return false
	}

	a = interval.Int{Cls: interval.LeftClosed, A: r.Y, B: r.Y + r.Height}
	b = interval.Int{Cls: interval.LeftClosed, A: s.Y, B: s.Y + s.Height}
	v0 := interval.Intersection(&a, &b)
	if v0.Class() == interval.Empty {
		return false
	}

	h := h0.(*interval.Int)
	v := v0.(*interval.Int)
	var y Rectangle
	y.X = h.A
	y.Y = v.A
	y.Width = h.B - h.A
	y.Height = v.B - v.A
	*r = y
	return true
}

func (r *Rectangle) join(s Rectangle) {
	if s.IsZero() {
		return
	}

	if r.IsZero() {
		*r = s
		return
	}

	r.X = mathutil.Min(r.X, s.X)
	r.Width = mathutil.Max(r.X+r.Width, s.X+s.Width) - r.X
	r.Y = mathutil.Min(r.Y, s.Y)
	r.Height = mathutil.Max(r.Y+r.Height, s.Y+s.Height) - r.Y
}

// Has returns whether r contains p.
func (r *Rectangle) Has(p Position) bool {
	return p.X >= r.X && p.X < r.X+r.Width &&
		p.Y >= r.Y && p.Y < r.Y+r.Height
}

// Size represents 2D dimensions.
type Size struct {
	Width, Height int
}

func newSize(w, h int) Size { return Size{w, h} }

// IsZero returns whether s.Width or s.Height is zero.
func (s *Size) IsZero() bool { return s.Width <= 0 || s.Height <= 0 }
