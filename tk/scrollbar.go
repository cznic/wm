// Copyright 2015 The WM Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tk

import (
	"github.com/cznic/mathutil"
	"github.com/cznic/wm"
	"github.com/gdamore/tcell"
)

// Scrollbar represents an UI element used to show that a View content
// overflows its window and provide visual feedback of the position of its
// viewport.
//
// Scrollbar methods must be called only directly from an event handler
// goroutine or from a function that was enqueued using wm.Application.Post or
// wm.Application.PostWait.
type Scrollbar struct {
	dragHandlePos0       int                          //
	dragScreenPos0       wm.Position                  //
	draggingHandle       bool                         //
	handlePos            int                          //
	handleSize           int                          //
	onClickDecrement     *wm.OnMouseHandlerList       //
	onClickDecrementPage *wm.OnMouseHandlerList       //
	onClickIncrement     *wm.OnMouseHandlerList       //
	onClickIncrementPage *wm.OnMouseHandlerList       //
	onPaint              *wm.OnPaintHandlerList       //
	onSetHandlePos       *wm.OnSetIntHandlerList      //
	onSetHandleSize      *wm.OnSetIntHandlerList      //
	onSetPosition        *wm.OnSetPositionHandlerList //
	onSetSize            *wm.OnSetSizeHandlerList     //
	onSetStyle           *wm.OnSetStyleHandlerList    //
	position             wm.Position                  //
	size                 wm.Size                      //
	style                wm.Style                     //
	w                    *wm.Window                   //
}

// NewScrollbar returns a newly created Scrollbar.
func NewScrollbar(w *wm.Window) *Scrollbar {
	s := &Scrollbar{w: w}
	s.OnPaint(s.onPaintHandler, nil)
	s.OnSetHandlePosition(s.onSetHandlePosHandler, nil)
	s.OnSetHandleSize(s.onSetHandleSizeHandler, nil)
	s.OnSetPosition(s.onSetPositionHandler, nil)
	s.OnSetSize(s.onSetSizeHandler, nil)
	s.OnSetStyle(s.onSetStyleHandler, nil)
	w.OnClickBorder(s.onClickBorderHandler, nil)
	w.OnClose(s.onCloseHandler, nil)
	w.OnDragBorder(s.onDragBorderHandler, nil)
	return s
}

func (s *Scrollbar) onCloseHandler(w *wm.Window, prev wm.OnCloseHandler) {
	if prev != nil {
		prev(w, nil)
	}
	s.onClickDecrement.Clear()
	s.onClickDecrementPage.Clear()
	s.onClickIncrement.Clear()
	s.onClickIncrementPage.Clear()
	s.onPaint.Clear()
	s.onSetHandlePos.Clear()
	s.onSetHandleSize.Clear()
	s.onSetPosition.Clear()
	s.onSetSize.Clear()
	s.onSetStyle.Clear()
}

const (
	decrementArrow = iota
	decrementPage
	scrollbarHandle
	incrementPage
	incrementArrow
)

func (s *Scrollbar) place(w *wm.Window, winPos wm.Position) int {
	pos := s.position
	sz := s.Size()
	switch {
	case s.isVertical():
		area := w.BorderRightArea()
		if !area.Clip(wm.Rectangle{Position: wm.Position{X: area.X + pos.X, Y: area.Y + pos.Y}, Size: sz}) || !winPos.In(area) {
			return -1
		}

		switch {
		case winPos.Y == pos.Y+sz.Height-1:
			return incrementArrow
		case winPos.Y == pos.Y:
			return decrementArrow
		case winPos.Y < pos.Y+1+s.HandlePosition():
			return decrementPage
		case winPos.Y > pos.Y+s.HandlePosition()+s.HandleSize():
			return incrementPage
		default:
			return scrollbarHandle
		}
	default:
		area := w.BorderBottomArea()
		if !area.Clip(wm.Rectangle{Position: wm.Position{X: area.X + pos.X, Y: area.Y + pos.Y}, Size: sz}) || !winPos.In(area) {
			return -1
		}

		switch {
		case winPos.X == pos.X+sz.Width-1:
			return incrementArrow
		case winPos.X == pos.X:
			return decrementArrow
		case winPos.X < pos.X+1+s.HandlePosition():
			return decrementPage
		case winPos.X > pos.X+s.HandlePosition()+s.HandleSize():
			return incrementPage
		default:
			return scrollbarHandle
		}
	}
}

func (s *Scrollbar) onMouseMoveHandler(w *wm.Window, prev wm.OnMouseHandler, button tcell.ButtonMask, screenPos, winPos wm.Position, mods tcell.ModMask) bool {
	if s.draggingHandle {
		switch {
		case s.isVertical():
			dy := screenPos.Y - s.dragScreenPos0.Y
			s.SetHandlePosition(s.dragHandlePos0 + dy)
		default:
			dx := screenPos.X - s.dragScreenPos0.X
			s.SetHandlePosition(s.dragHandlePos0 + dx)
		}
		return true
	}

	return prev != nil && prev(w, nil, button, screenPos, winPos, mods)
}

func (s *Scrollbar) onDropHandler(w *wm.Window, prev wm.OnMouseHandler, button tcell.ButtonMask, screenPos, winPos wm.Position, mods tcell.ModMask) bool {
	if s.draggingHandle {
		r := s.w.Desktop().Root()
		r.RemoveOnDrop()
		r.RemoveOnMouseMove()
		s.w.BringToFront()
		s.w.SetFocus(true)
		if w := s.w; w != r {
			w.RemoveOnDrop()
			w.RemoveOnMouseMove()
		}
		s.draggingHandle = false
		return true
	}

	return prev != nil && prev(w, nil, button, screenPos, winPos, mods)
}

func (s *Scrollbar) onDragBorderHandler(w *wm.Window, prev wm.OnMouseHandler, button tcell.ButtonMask, screenPos, winPos wm.Position, mods tcell.ModMask) bool {
	if prev != nil && prev(w, nil, button, screenPos, winPos, mods) {
		return true
	}

	if button != tcell.Button1 || mods != 0 {
		return false
	}

	switch s.place(w, winPos) {
	case scrollbarHandle:
		s.draggingHandle = true
		r := s.w.Desktop().Root()
		r.OnDrop(s.onDropHandler, nil)
		r.OnMouseMove(s.onMouseMoveHandler, nil)
		s.dragHandlePos0 = s.HandlePosition()
		s.dragScreenPos0 = screenPos
		if w := s.w; w != r {
			w.OnDrop(s.onDropHandler, nil)
			w.OnMouseMove(s.onMouseMoveHandler, nil)
		}
		s.w.BringToFront()
		s.w.SetFocus(true)
		return true
	default:
		return false
	}

}

func (s *Scrollbar) onClickBorderHandler(w *wm.Window, prev wm.OnMouseHandler, button tcell.ButtonMask, screenPos, winPos wm.Position, mods tcell.ModMask) bool {
	if prev != nil && prev(w, nil, button, screenPos, winPos, mods) {
		return true
	}

	if button != tcell.Button1 || mods != 0 {
		return false
	}

	switch s.place(w, winPos) {
	case decrementArrow:
		return s.onClickDecrement.Handle(w, button, screenPos, winPos, mods)
	case decrementPage:
		return s.onClickDecrementPage.Handle(w, button, screenPos, winPos, mods)
	case incrementPage:
		return s.onClickIncrementPage.Handle(w, button, screenPos, winPos, mods)
	case incrementArrow:
		return s.onClickIncrement.Handle(w, button, screenPos, winPos, mods)
	default:
		return false
	}
}

func (s *Scrollbar) onSetHandlePosHandler(w *wm.Window, prev wm.OnSetIntHandler, dst *int, src int) {
	if prev != nil {
		panic("internal error")
	}

	sz := s.Size().Width - 2
	if s.isVertical() {
		sz = s.Size().Height - 2
	}
	src = mathutil.Max(0, mathutil.Min(sz-s.HandleSize(), src))
	*dst = src
	w.Invalidate(w.Area())
}

func (s *Scrollbar) onSetHandleSizeHandler(w *wm.Window, prev wm.OnSetIntHandler, dst *int, src int) {
	if prev != nil {
		panic("internal error")
	}

	sz := s.Size().Width - 2
	if s.isVertical() {
		sz = s.Size().Height - 2
	}
	src = mathutil.Max(0, mathutil.Min(sz-s.HandlePosition(), src))
	*dst = src
	w.Invalidate(w.Area())
}

func (s *Scrollbar) onSetPositionHandler(w *wm.Window, prev wm.OnSetPositionHandler, dst *wm.Position, src wm.Position) {
	if prev != nil {
		panic("internal error")
	}

	*dst = src
	w.Invalidate(w.Area())
}

func (s *Scrollbar) onSetSizeHandler(w *wm.Window, prev wm.OnSetSizeHandler, dst *wm.Size, src wm.Size) {
	if prev != nil {
		panic("internal error")
	}

	*dst = src
	w.Invalidate(w.Area())
}

func (s *Scrollbar) onSetStyleHandler(w *wm.Window, prev wm.OnSetStyleHandler, dst *wm.Style, src wm.Style) {
	if prev != nil {
		panic("internal error")
	}

	*dst = src
	w.Invalidate(w.Area())
}

func (s *Scrollbar) onPaintHandler(w *wm.Window, prev wm.OnPaintHandler, ctx wm.PaintContext) {
	if prev != nil {
		panic("internal error")
	}

	sz := s.Size()
	pos := s.Position()
	style := s.Style().TCellStyle()
	switch {
	case s.isVertical():
		if w.ClientSize().Width == 0 {
			break
		}

		origin := pos.Y
		for y := 1; y < sz.Height-1; y++ {
			w.SetCell(pos.X, origin+y, tcell.RuneBoard, nil, style)
		}
		if sz.Height < 2 {
			break
		}

		for y := 1 + s.handlePos; y < 1+s.handlePos+s.handleSize && y < sz.Height-1; y++ {
			w.SetCell(pos.X, origin+y, tcell.RuneCkBoard, nil, style)
		}
		w.SetCell(pos.X, origin, '▴', nil, style)
		w.SetCell(pos.X, origin+sz.Height-1, '▾', nil, style)
	default:
		if w.ClientSize().Height == 0 {
			break
		}

		origin := pos.X
		for x := 1; x < sz.Width-1; x++ {
			w.SetCell(origin+x, pos.Y, tcell.RuneBoard, nil, style)
		}
		if sz.Width < 2 {
			break
		}

		for x := 1 + s.handlePos; x < 1+s.handlePos+s.handleSize && x < sz.Width-1; x++ {
			w.SetCell(origin+x, pos.Y, tcell.RuneCkBoard, nil, style)
		}

		w.SetCell(origin, pos.Y, '◂', nil, style)
		w.SetCell(origin+sz.Width-1, pos.Y, '▸', nil, style)
	}
}

func (s *Scrollbar) isVertical() bool { return s.Size().Width == 1 }

// ----------------------------------------------------------------------------

// HandlePosition returns the position of the scrollbar handle.
func (s *Scrollbar) HandlePosition() int { return s.handlePos }

// HandleSize returns the size of the scrollbar handle.
func (s *Scrollbar) HandleSize() int { return s.handleSize }

// OnClickIncrement sets a handler invokend on clicking the right arrow of a
// horizontal scrollbar or the down arrow of a vertical scrollbar. When the
// event handler is removed, finalize is called, if not nil.
func (s *Scrollbar) OnClickIncrement(h wm.OnMouseHandler, finalize func()) {
	wm.AddOnMouseHandler(&s.onClickIncrement, h, finalize)
}

// OnClickIncrementPage sets a handler invokend on clicking the area between
// the scrollbar handle and the right arrow of a horizontal scrollbar or
// between the scrollbar handle an dthe down arrow of a vertical scrollbar.
// When the event handler is removed, finalize is called, if not nil.
func (s *Scrollbar) OnClickIncrementPage(h wm.OnMouseHandler, finalize func()) {
	wm.AddOnMouseHandler(&s.onClickIncrementPage, h, finalize)
}

// OnClickDecrement sets a handler invokend on clicking the left arrow of a
// horizontal scrollbar or the up arrow of a vertical scrollbar. When the
// event handler is removed, finalize is called, if not nil.
func (s *Scrollbar) OnClickDecrement(h wm.OnMouseHandler, finalize func()) {
	wm.AddOnMouseHandler(&s.onClickDecrement, h, finalize)
}

// OnClickDecrementPage sets a handler invokend on clicking the area between
// the left arrow of and the scrollbar handle of a horizontal scrollbar or
// between the the up arrow and the scrollbar handle of a vertical scrollbar.
// When the event handler is removed, finalize is called, if not nil.
func (s *Scrollbar) OnClickDecrementPage(h wm.OnMouseHandler, finalize func()) {
	wm.AddOnMouseHandler(&s.onClickDecrementPage, h, finalize)
}

// OnPaint sets a paint handler. When the event handler is removed, finalize is
// called, if not nil.
func (s *Scrollbar) OnPaint(h wm.OnPaintHandler, finalize func()) {
	wm.AddOnPaintHandler(&s.onPaint, h, finalize)
}

// OnSetHandlePosition sets a handler invoked on SetHandlePosition. When the
// event handler is removed, finalize is called, if not nil.
func (s *Scrollbar) OnSetHandlePosition(h wm.OnSetIntHandler, finalize func()) {
	wm.AddOnSetIntHandler(&s.onSetHandlePos, h, finalize)
}

// OnSetHandleSize sets a handler invoked on SetHandleSize. When the
// event handler is removed, finalize is called, if not nil.
func (s *Scrollbar) OnSetHandleSize(h wm.OnSetIntHandler, finalize func()) {
	wm.AddOnSetIntHandler(&s.onSetHandleSize, h, finalize)
}

// OnSetPosition sets a handler invoked on SetPosition. When the event handler
// is removed, finalize is called, if not nil.
func (s *Scrollbar) OnSetPosition(h wm.OnSetPositionHandler, finalize func()) {
	wm.AddOnSetPositionHandler(&s.onSetPosition, h, finalize)
}

// OnSetSize sets a handler invoked on SetSize. When the event handler
// is removed, finalize is called, if not nil.
func (s *Scrollbar) OnSetSize(h wm.OnSetSizeHandler, finalize func()) {
	wm.AddOnSetSizeHandler(&s.onSetSize, h, finalize)
}

// OnSetStyle sets a handler invoked on SetStyle. When the event handler
// is removed, finalize is called, if not nil.
func (s *Scrollbar) OnSetStyle(h wm.OnSetStyleHandler, finalize func()) {
	wm.AddOnSetStyleHandler(&s.onSetStyle, h, finalize)
}

// Paint ask the scrollbar to render itself. It is intended to be only called
// from a "hook" paint handler that determines where the scrollbar appears.
func (s *Scrollbar) Paint(ctx wm.PaintContext) { s.onPaint.Handle(s.w, ctx) }

// Position returns the position of the scrollbar.
func (s *Scrollbar) Position() wm.Position { return s.position }

// RemoveOnClickIncrement undoes the most recent OnClickIncrement call. The
// function will panic if there is no handler set.
func (s *Scrollbar) RemoveOnClickIncrement() { wm.RemoveOnMouseHandler(&s.onClickIncrement) }

// RemoveOnClickIncrementPage undoes the most recent OnClickIncrementPage call.
// The function will panic if there is no handler set.
func (s *Scrollbar) RemoveOnClickIncrementPage() { wm.RemoveOnMouseHandler(&s.onClickIncrementPage) }

// RemoveOnClickDecrement undoes the most recent OnClickDecrement call. The
// function will panic if there is no handler set.
func (s *Scrollbar) RemoveOnClickDecrement() { wm.RemoveOnMouseHandler(&s.onClickDecrement) }

// RemoveOnClickDecrementPage undoes the most recent OnClickDecrement call. The
// function will panic if there is no handler set.
func (s *Scrollbar) RemoveOnClickDecrementPage() { wm.RemoveOnMouseHandler(&s.onClickDecrementPage) }

// RemoveOnPaint undoes the most recent OnPaint call. The function will panic
// if there is no handler set.
func (s *Scrollbar) RemoveOnPaint() { wm.RemoveOnPaintHandler(&s.onPaint) }

// RemoveOnSetHandlePosition undoes the most recent OnSetHandlePosition call.
// The function will panic if there is no handler set.
func (s *Scrollbar) RemoveOnSetHandlePosition() { wm.RemoveOnSetIntHandler(&s.onSetHandlePos) }

// RemoveOnSetHandleSize undoes the most recent OnSetHandleSize call.
// The function will panic if there is no handler set.
func (s *Scrollbar) RemoveOnSetHandleSize() { wm.RemoveOnSetIntHandler(&s.onSetHandleSize) }

// RemoveOnSetPosition undoes the most recent OnSetPosition call. The function
// will panic if there is no handler set.
func (s *Scrollbar) RemoveOnSetPosition() { wm.RemoveOnSetPositionHandler(&s.onSetPosition) }

// RemoveOnSetSize undoes the most recent OnSetSize call. The function
// will panic if there is no handler set.
func (s *Scrollbar) RemoveOnSetSize() { wm.RemoveOnSetSizeHandler(&s.onSetSize) }

// RemoveOnSetStyle undoes the most recent OnSetStyle call. The function
// will panic if there is no handler set.
func (s *Scrollbar) RemoveOnSetStyle() { wm.RemoveOnSetStyleHandler(&s.onSetStyle) }

// SetPosition sets the scrollbar position.
func (s *Scrollbar) SetPosition(v wm.Position) { s.onSetPosition.Handle(s.w, &s.position, v) }

// SetSize sets the scrollbar size.
func (s *Scrollbar) SetSize(v wm.Size) { s.onSetSize.Handle(s.w, &s.size, v) }

// Size returns the size of the scrollbar.
func (s *Scrollbar) Size() wm.Size { return s.size }

// SetHandlePosition sets the scrollbar handle position.
func (s *Scrollbar) SetHandlePosition(v int) { s.onSetHandlePos.Handle(s.w, &s.handlePos, v) }

// SetHandleSize sets the scrollbar handle size.
func (s *Scrollbar) SetHandleSize(v int) { s.onSetHandleSize.Handle(s.w, &s.handleSize, v) }

// SetStyle sets the scrollbar style.
func (s *Scrollbar) SetStyle(v wm.Style) { s.onSetStyle.Handle(s.w, &s.style, v) }

// SetView sets the scrollbar parameters based on the view parameters. SetView panics when origin < 0.
func (s *Scrollbar) SetView(origin, viewportSize, contentSize int) {
	if origin < 0 {
		panic("Scrollbar.SetView: invalid origin")
	}

	if contentSize < 1 { // Unknown content size.
		s.SetHandlePosition(0)
		s.SetHandleSize(0)
		s.w.Invalidate(s.w.Area())
		return
	}

	clip := wm.NewRectangle(-origin, 0, contentSize, 0)
	clip.Clip(wm.NewRectangle(0, 0, viewportSize, 0))

	scrollbarSize := s.size.Width - 2 // Sans arrows.
	if s.isVertical() {
		scrollbarSize = s.size.Height - 2
	}
	handlePos := mathutil.Min(scrollbarSize-1, (origin*scrollbarSize+contentSize/2)/contentSize)
	handleSize := mathutil.Max(1, clip.Width*scrollbarSize/contentSize)
	s.SetHandlePosition(handlePos)
	s.SetHandleSize(handleSize)
	s.w.Invalidate(s.w.Area())
}

// Style returns the style of the scrollbar.
func (s *Scrollbar) Style() wm.Style { return s.style }
