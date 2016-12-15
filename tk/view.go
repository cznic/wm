// Copyright 2016 The WM Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tk

import (
	"github.com/cznic/mathutil"
	"github.com/cznic/wm"
	"github.com/gdamore/tcell"
)

// Meter provides metrics of content displayed in the client area of a window.
type Meter interface {
	// Metrics is called when window's viewport is set or updated.  The
	// result .Height reflects the total contents height. The result .Width
	// reflect the maximum width of the content in area.  If any of the
	// values are unknown a negative number is returned in the respective
	// output variable.
	Metrics(viewport wm.Rectangle) wm.Size
}

// View displays content possibly overflowing the size of its client area.
//
// View methods must be called only directly from an event handler goroutine or
// from a function that was enqueued using wm.Application.Post or
// wm.Application.PostWait.
type View struct {
	*wm.Window     // Underlying window.
	hs             *Scrollbar
	hsEnabled      bool
	hsShown        bool
	meter          Meter
	metrics        wm.Size
	onSetHSEnabled *wm.OnSetBoolHandlerList
	onSetVSEnabled *wm.OnSetBoolHandlerList
	updating       bool
	vs             *Scrollbar
	vsEnabled      bool
	vsShown        bool
}

// NewView configures w to show scrollbars when content, measured using the
// meter parameter, overflows the client area of w and returns the resulting
// View.  The user can use the scrollbars to control the View's Origin.
//
// NewView must be called only directly from an event handler goroutine or from
// a function that was enqueued using wm.Application.Post or
// wm.Application.PostWait.
func NewView(w *wm.Window, meter Meter) *View {
	vs := NewScrollbar(w)
	vs.SetStyle(wm.Style{Background: tcell.ColorSilver, Foreground: tcell.ColorBlack})
	hs := NewScrollbar(w)
	hs.SetStyle(vs.Style())
	v := &View{
		Window:    w,
		hs:        hs,
		hsEnabled: true,
		meter:     meter,
		vs:        vs,
		vsEnabled: true,
	}
	hs.OnClickDecrement(v.onClickDecrementHS, nil)
	hs.OnClickDecrementPage(v.onClickDecrementHSPage, nil)
	hs.OnClickIncrement(v.onClickIncrementHS, nil)
	hs.OnClickIncrementPage(v.onClickIncrementHSPage, nil)
	hs.OnSetHandlePosition(v.onSetHandlePositionHS, nil)
	v.OnSetHorizontalScrollbarEnabled(v.onSetHorizontalScrollbarEnabledHandler, nil)
	v.OnSetVerticalScrollbarEnabled(v.onSetVerticalScrollbarEnabledHandler, nil)
	vs.OnClickDecrement(v.onClickDecrementVS, nil)
	vs.OnClickDecrementPage(v.onClickDecrementVSPage, nil)
	vs.OnClickIncrement(v.onClickIncrementVS, nil)
	vs.OnClickIncrementPage(v.onClickIncrementVSPage, nil)
	vs.OnSetHandlePosition(v.onSetHandlePositionVS, nil)
	w.OnClose(v.onCloseHandler, nil)
	w.OnMouseMove(v.onMouseMoveHandler, nil)
	w.OnPaintBorderBottom(v.onPaintBorderBottomHandler, nil)
	w.OnPaintBorderRight(v.onPaintBorderRightHandler, nil)
	w.OnSetClientSize(v.onSetClientSizeHandler, nil)
	w.OnSetOrigin(v.onSetOriginHandler, nil)
	return v
}

func (v *View) onCloseHandler(w *wm.Window, prev wm.OnCloseHandler) {
	if prev != nil {
		prev(w, nil)
	}
	v.onSetHSEnabled.Clear()
	v.onSetVSEnabled.Clear()
}

func (v *View) onMouseMoveHandler(w *wm.Window, prev wm.OnMouseHandler, button tcell.ButtonMask, screenPos, winPos wm.Position, mods tcell.ModMask) bool {
	if prev != nil && prev(w, nil, button, screenPos, winPos, mods) {
		return true
	}

	switch button {
	case tcell.WheelLeft:
		o := v.Origin()
		o.X = mathutil.Max(0, o.X-1)
		v.SetOrigin(o)
		return true
	case tcell.WheelRight:
		o := v.Origin()
		o.X++
		v.SetOrigin(o)
		return true
	case tcell.WheelUp:
		o := v.Origin()
		o.Y = mathutil.Max(0, o.Y-1)
		v.SetOrigin(o)
		return true
	case tcell.WheelDown:
		o := v.Origin()
		o.Y++
		v.SetOrigin(o)
		return true
	default:
		return false
	}
}

func (v *View) onClickDecrementHSPage(w *wm.Window, prev wm.OnMouseHandler, button tcell.ButtonMask, screenPos, winPos wm.Position, mods tcell.ModMask) bool {
	if !v.hsShown {
		return false
	}

	o := v.Origin()
	o.X = mathutil.Max(0, o.X-v.ClientSize().Width)
	v.SetOrigin(o)
	return true
}

func (v *View) onClickIncrementHSPage(w *wm.Window, prev wm.OnMouseHandler, button tcell.ButtonMask, screenPos, winPos wm.Position, mods tcell.ModMask) bool {
	if !v.hsShown {
		return false
	}

	o := v.Origin()
	o.X += v.ClientSize().Width
	v.SetOrigin(o)
	return true
}

func (v *View) onSetHandlePositionHS(w *wm.Window, prev wm.OnSetIntHandler, dst *int, src int) {
	if prev != nil {
		prev(w, nil, dst, src)
		src = *dst
	}

	if !v.hs.draggingHandle || v.updating {
		return
	}

	if v.metrics.Width < 0 {
		return
	}

	if src+v.hs.HandleSize() == v.hs.Size().Width-2 {
		v.SetOrigin(wm.Position{X: v.metrics.Width - v.ClientArea().Width, Y: v.Origin().Y})
		return
	}

	x := (2*v.metrics.Width*src - v.metrics.Width) / (2*v.hs.Size().Width - 4)
	v.SetOrigin(wm.Position{X: x, Y: v.Origin().Y})
}

func (v *View) onSetHandlePositionVS(w *wm.Window, prev wm.OnSetIntHandler, dst *int, src int) {
	if prev != nil {
		prev(w, nil, dst, src)
		src = *dst
	}

	if !v.vs.draggingHandle || v.updating {
		return
	}

	if v.metrics.Height < 0 {
		return
	}

	if src+v.vs.HandleSize() == v.vs.Size().Height-2 {
		v.SetOrigin(wm.Position{X: v.Origin().X, Y: v.metrics.Height - v.ClientArea().Height})
		return
	}

	y := (2*v.metrics.Height*src - v.metrics.Height) / (2*v.vs.Size().Height - 4)
	v.SetOrigin(wm.Position{X: v.Origin().X, Y: y})
}

func (v *View) onClickDecrementVSPage(w *wm.Window, prev wm.OnMouseHandler, button tcell.ButtonMask, screenPos, winPos wm.Position, mods tcell.ModMask) bool {
	if !v.vsShown {
		return false
	}

	v.PageUp()
	return true
}

func (v *View) onClickIncrementVSPage(w *wm.Window, prev wm.OnMouseHandler, button tcell.ButtonMask, screenPos, winPos wm.Position, mods tcell.ModMask) bool {
	if !v.vsShown {
		return false
	}

	v.PageDown()
	return true
}

func (v *View) onClickDecrementHS(w *wm.Window, prev wm.OnMouseHandler, button tcell.ButtonMask, screenPos, winPos wm.Position, mods tcell.ModMask) bool {
	if !v.hsShown {
		return false
	}

	o := v.Origin()
	o.X = mathutil.Max(0, o.X-1)
	v.SetOrigin(o)
	return true
}

func (v *View) onClickIncrementHS(w *wm.Window, prev wm.OnMouseHandler, button tcell.ButtonMask, screenPos, winPos wm.Position, mods tcell.ModMask) bool {
	if !v.hsShown {
		return false
	}

	o := v.Origin()
	o.X++
	v.SetOrigin(o)
	return true
}

func (v *View) onClickDecrementVS(w *wm.Window, prev wm.OnMouseHandler, button tcell.ButtonMask, screenPos, winPos wm.Position, mods tcell.ModMask) bool {
	if !v.vsShown {
		return false
	}

	o := v.Origin()
	o.Y = mathutil.Max(0, o.Y-1)
	v.SetOrigin(o)
	return true
}

func (v *View) onClickIncrementVS(w *wm.Window, prev wm.OnMouseHandler, button tcell.ButtonMask, screenPos, winPos wm.Position, mods tcell.ModMask) bool {
	if !v.vsShown {
		return false
	}

	o := v.Origin()
	o.Y++
	v.SetOrigin(o)
	return true
}

func (v *View) onPaintBorderRightHandler(w *wm.Window, prev wm.OnPaintHandler, ctx wm.PaintContext) {
	if prev != nil {
		prev(w, nil, ctx)
	}
	v.vs.Paint(ctx)
}

func (v *View) onPaintBorderBottomHandler(w *wm.Window, prev wm.OnPaintHandler, ctx wm.PaintContext) {
	if prev != nil {
		prev(w, nil, ctx)
	}
	v.hs.Paint(ctx)
}

func (v *View) onSetOriginHandler(w *wm.Window, prev wm.OnSetPositionHandler, dst *wm.Position, src wm.Position) {
	if w := v.metrics.Width; w >= 0 {
		src.X = mathutil.Max(0, mathutil.Min(src.X, w-v.ClientSize().Width))
	}
	if h := v.metrics.Height; h >= 0 {
		src.Y = mathutil.Max(0, mathutil.Min(src.Y, h-v.ClientSize().Height))
	}

	if prev != nil {
		prev(w, nil, dst, src)
		src = *dst
	}
	*dst = src
	v.updateScrollBars()
}

func (v *View) onSetClientSizeHandler(w *wm.Window, prev wm.OnSetSizeHandler, dst *wm.Size, src wm.Size) {
	if prev != nil {
		prev(w, nil, dst, src)
	}
	*dst = src
	v.updateScrollBars()
}

func (v *View) onSetHorizontalScrollbarEnabledHandler(w *wm.Window, prev wm.OnSetBoolHandler, dst *bool, src bool) {
	if prev != nil {
		panic("internal error")
	}

	*dst = src
	v.updateScrollBars()
}

func (v *View) onSetVerticalScrollbarEnabledHandler(w *wm.Window, prev wm.OnSetBoolHandler, dst *bool, src bool) {
	if prev != nil {
		panic("internal error")
	}

	*dst = src
	v.updateScrollBars()
}

func checkHS(sz wm.Size, viewport wm.Rectangle) bool {
	return viewport.Width >= 2 && (viewport.X != 0 && sz.Width > 0 || sz.Width > viewport.Width || sz.Width < 0)
}

func checkVS(sz wm.Size, viewport wm.Rectangle) bool {
	return viewport.Height >= 2 && (viewport.Y != 0 && sz.Height > 0 || sz.Height > viewport.Height || sz.Height < 0)
}

func (v *View) updateScrollBars() {
	if v.updating {
		return
	}

	v.updating = true
	if v.hsShown {
		v.hs.SetPosition(wm.Position{Y: -1})
		v.SetBorderBottom(v.BorderBottom() - 1)
	}
	if v.vsShown {
		v.vs.SetPosition(wm.Position{X: -1})
		v.SetBorderRight(v.BorderRight() - 1)
	}

	viewport := v.ClientArea()
	viewport.Position = v.Origin()
	v.metrics = v.meter.Metrics(viewport)
	var showHS, showVS bool
	if showHS = v.hsEnabled && checkHS(v.metrics, viewport); showHS {
		viewport.Height--
		showVS = v.vsEnabled && checkVS(v.metrics, viewport)
	} else if showVS = v.vsEnabled && checkVS(v.metrics, viewport); showVS {
		viewport.Width--
		showHS = v.hsEnabled && checkHS(v.metrics, viewport)
	}

	if showHS {
		v.SetBorderBottom(v.BorderBottom() + 1)
	}
	if showVS {
		v.SetBorderRight(v.BorderRight() + 1)
	}

	cla := v.ClientArea()
	if showHS {
		v.hs.SetSize(wm.Size{Width: cla.Width, Height: 1})
		v.hs.SetPosition(wm.Position{X: v.BorderLeft()})
		v.hs.SetView(v.Origin().X, cla.Width, v.metrics.Width)
	}
	if showVS {
		v.vs.SetSize(wm.Size{Width: 1, Height: cla.Height})
		v.vs.SetPosition(wm.Position{Y: v.BorderTop()})
		v.vs.SetView(v.Origin().Y, cla.Height, v.metrics.Height)
	}

	v.hsShown = showHS
	v.vsShown = showVS
	v.updating = false
}

// ----------------------------------------------------------------------------

// HorizontalScrollbarEnabled reports whether the horizontal scrollbar is
// enabled.
func (v *View) HorizontalScrollbarEnabled() bool { return v.hsEnabled }

// SetHorizontalScrollbarEnabled sets whether the horizontal scrollbar is
// enabled.
func (v *View) SetHorizontalScrollbarEnabled(b bool) {
	v.onSetHSEnabled.Handle(v.Window, &v.hsEnabled, b)
}

// OnSetHorizontalScrollbarEnabled sets a handler invoked on
// SetHorizontalScrollbarEnabled. When the event handler is removed, finalize
// is called, if not nil.
func (v *View) OnSetHorizontalScrollbarEnabled(h wm.OnSetBoolHandler, finalize func()) {
	wm.AddOnSetBoolHandler(&v.onSetHSEnabled, h, finalize)
}

// RemoveOnSetHorizontalScrollbarEnabled undoes the most recent
// OnSetHorizontalScrollbarEnabled call.  The function will panic if there is
// no handler set.
func (v *View) RemoveOnSetHorizontalScrollbarEnabled() { wm.RemoveOnSetBoolHandler(&v.onSetHSEnabled) }

// VerticalScrollbarEnabled reports whether the vertical scrollbar is enabled.
func (v *View) VerticalScrollbarEnabled() bool { return v.vsEnabled }

// SetVerticalScrollbarEnabled sets whether the vertical scrollbar is enabled.
func (v *View) SetVerticalScrollbarEnabled(b bool) { v.onSetVSEnabled.Handle(v.Window, &v.vsEnabled, b) }

// OnSetVerticalScrollbarEnabled sets a handler invoked on
// SetVerticalScrollbarEnabled. When the event handler is removed, finalize is
// called, if not nil.
func (v *View) OnSetVerticalScrollbarEnabled(h wm.OnSetBoolHandler, finalize func()) {
	wm.AddOnSetBoolHandler(&v.onSetVSEnabled, h, finalize)
}

// RemoveOnSetVerticalScrollbarEnabled undoes the most recent
// OnSetVerticalScrollbarEnabled call.  The function will panic if there is no
// handler set.
func (v *View) RemoveOnSetVerticalScrollbarEnabled() { wm.RemoveOnSetBoolHandler(&v.onSetVSEnabled) }

// Home makes the view show the beginning of its content.
func (v *View) Home() { v.SetOrigin(wm.Position{}) }

// End makes the view show the ending of its content.
func (v *View) End() {
	if m := v.meter.Metrics(wm.Rectangle{Size: wm.Size{Width: 1, Height: 1}}); m.Height >= 0 {
		v.SetOrigin(wm.Position{Y: m.Height - v.ClientArea().Height})
	}
}

// PageDown makes the view show the next page of content.
func (v *View) PageDown() {
	o := v.Origin()
	o.Y += v.ClientSize().Height
	v.SetOrigin(o)
}

// PageUp makes the view show the previous page of content.
func (v *View) PageUp() {
	o := v.Origin()
	o.Y -= v.ClientSize().Height
	v.SetOrigin(o)
}
