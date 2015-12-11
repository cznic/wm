// Copyright 2015 The WM Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wm

import (
	"github.com/gdamore/tcell"
	"time"
)

// PaintContext represent painting context passed to paint handlers.
type PaintContext struct {
	Rectangle
	origin Position
}

// OnCloseHandler is called on window close. If there was a previous handler
// installed, it's passed in prev. The handler then has the opportunity to call
// the previous handler before or after its own execution.
type OnCloseHandler func(w *Window, prev OnCloseHandler)

type onCloseHandlerList struct {
	prev      *onCloseHandlerList
	h         OnCloseHandler
	finalizer func()
}

func addOnCloseHandler(w *Window, l **onCloseHandlerList, h OnCloseHandler, finalizer func()) {
	prev := *l
	if prev == nil {
		*l = &onCloseHandlerList{
			h:         h,
			finalizer: finalizer,
		}
		return
	}

	*l = &onCloseHandlerList{
		prev: prev,
		h: func(_ *Window, _ OnCloseHandler) {
			h(w, prev.h)
		},
		finalizer: finalizer,
	}
}

func (l *onCloseHandlerList) clear() {
	for l != nil {
		if f := l.finalizer; f != nil {
			f()
		}
		l = l.prev
	}
}

func (l *onCloseHandlerList) handle(w *Window) {
	if l != nil {
		w.beginUpdate()
		l.h(w, nil)
		w.endUpdate()
		return
	}

}

func removeOnCloseHandler(w *Window, l **onCloseHandlerList) {
	node := *l
	*l = node.prev
	if f := node.finalizer; f != nil {
		f()
	}
}

// OnKeyHandler handles key events. If there was a previous handler installed,
// it's passed in prev. The handler then has the opportunity to call the
// previous handler before or after its own execution.  The handler should
// return true if it consumes the event and it should not be considered by
// other subscribed handlers.
//
type OnKeyHandler func(w *Window, prev OnKeyHandler, key tcell.Key, mod tcell.ModMask, r rune) bool

type onKeyHandlerList struct {
	prev      *onKeyHandlerList
	h         OnKeyHandler
	finalizer func()
}

func addOnKeyHandler(w *Window, l **onKeyHandlerList, h OnKeyHandler, finalizer func()) {
	prev := *l
	if prev == nil {
		*l = &onKeyHandlerList{
			h:         h,
			finalizer: finalizer,
		}
		return
	}

	*l = &onKeyHandlerList{
		prev: prev,
		h: func(_ *Window, _ OnKeyHandler, key tcell.Key, mod tcell.ModMask, r rune) bool {
			return h(w, prev.h, key, mod, r)
		},
		finalizer: finalizer,
	}
}

func (l *onKeyHandlerList) clear() {
	for l != nil {
		if f := l.finalizer; f != nil {
			f()
		}
		l = l.prev
	}
}

func (l *onKeyHandlerList) handle(w *Window, key tcell.Key, mod tcell.ModMask, r rune) {
	if l != nil {
		w.beginUpdate()
		l.h(w, nil, key, mod, r)
		w.endUpdate()
		return
	}

}

func removeOnKeyHandler(w *Window, l **onKeyHandlerList) {
	node := *l
	*l = node.prev
	if f := node.finalizer; f != nil {
		f()
	}
}

// OnMouseHandler handles mouse events. If there was a previous handler
// installed, it's passed in prev. The handler then has the opportunity to call
// the previous handler before or after its own execution. The handler should
// return true if it consumes the event and it should not be considered by
// other subscribed handlers.
type OnMouseHandler func(w *Window, prev OnMouseHandler, button tcell.ButtonMask, screenPos, winPos Position, mods tcell.ModMask) bool

type onMouseHandlerList struct {
	prev      *onMouseHandlerList
	h         OnMouseHandler
	finalizer func()
}

func addOnMouseHandler(w *Window, l **onMouseHandlerList, h OnMouseHandler, finalizer func()) {
	prev := *l
	if prev == nil {
		*l = &onMouseHandlerList{
			h:         h,
			finalizer: finalizer,
		}
		return
	}

	*l = &onMouseHandlerList{
		prev: prev,
		h: func(_ *Window, _ OnMouseHandler, button tcell.ButtonMask, screenPos, winPos Position, mods tcell.ModMask) bool {
			return h(w, prev.h, button, screenPos, winPos, mods)
		},
		finalizer: finalizer,
	}
}

func (l *onMouseHandlerList) clear() {
	for l != nil {
		if f := l.finalizer; f != nil {
			f()
		}
		l = l.prev
	}
}

func (l *onMouseHandlerList) handle(w *Window, button tcell.ButtonMask, screenPos, winPos Position, mods tcell.ModMask) {
	if l != nil {
		w.beginUpdate()
		l.h(w, nil, button, screenPos, winPos, mods)
		w.endUpdate()
	}
}

func removeOnMouseHandler(w *Window, l **onMouseHandlerList) {
	node := *l
	*l = node.prev
	if f := node.finalizer; f != nil {
		f()
	}
}

// OnPaintHandler handles paint requests. If there was a previous handler
// installed, it's passed in prev. The handler then has the opportunity to call
// the previous handler before or after its own execution.
//
// OnPaint handlers can safely try to modify window cells outside of the area
// argument as such attempts will be silently ignored. In other words, an
// OnPaintHandler cannot affect any window cell outside of the area argument.
type OnPaintHandler func(w *Window, prev OnPaintHandler, ctx PaintContext)

type onPaintHandlerList struct {
	prev      *onPaintHandlerList
	h         OnPaintHandler
	finalizer func()
}

func addOnPaintHandler(w *Window, l **onPaintHandlerList, h OnPaintHandler, finalizer func()) {
	prev := *l
	if prev == nil {
		*l = &onPaintHandlerList{
			h: func(_ *Window, _ OnPaintHandler, ctx PaintContext) {
				saveArea, saveOrigin := w.setPaintContext(ctx.Rectangle, ctx.origin)
				h(w, nil, ctx)
				w.setPaintContext(saveArea, saveOrigin)
			},
			finalizer: finalizer,
		}
		return
	}

	*l = &onPaintHandlerList{
		prev: prev,
		h: func(_ *Window, _ OnPaintHandler, ctx PaintContext) {
			saveArea, saveOrigin := w.setPaintContext(ctx.Rectangle, ctx.origin)
			h(w, prev.h, ctx)
			w.setPaintContext(saveArea, saveOrigin)
		},
		finalizer: finalizer,
	}
}

func (l *onPaintHandlerList) clear() {
	for l != nil {
		if f := l.finalizer; f != nil {
			f()
		}
		l = l.prev
	}
}

func (l *onPaintHandlerList) handle(w *Window, ctx PaintContext) {
	if l == nil || ctx.Rectangle.IsZero() {
		return
	}

	l.h(w, nil, ctx)
}

func removeOnPaintHandler(w *Window, l **onPaintHandlerList) {
	node := *l
	*l = node.prev
	if f := node.finalizer; f != nil {
		f()
	}
}

// OnSetDesktopHandler handles requests to change values of type *Desktop. If
// there was a previous handler installed, it's passed in prev. The handler
// then has the opportunity to call the previous handler before or after its
// own execution.
type OnSetDesktopHandler func(w *Window, prev OnSetDesktopHandler, dst **Desktop, src *Desktop)

type onSetDesktopHandlerList struct {
	prev      *onSetDesktopHandlerList
	h         OnSetDesktopHandler
	finalizer func()
}

func addOnSetDesktopHandler(w *Window, l **onSetDesktopHandlerList, h OnSetDesktopHandler, finalizer func()) {
	prev := *l
	if prev == nil {
		*l = &onSetDesktopHandlerList{
			h:         h,
			finalizer: finalizer,
		}
		return
	}

	*l = &onSetDesktopHandlerList{
		prev: prev,
		h: func(_ *Window, _ OnSetDesktopHandler, dst **Desktop, src *Desktop) {
			h(w, prev.h, dst, src)
		},
		finalizer: finalizer,
	}
}

func (l *onSetDesktopHandlerList) clear() {
	for l != nil {
		if f := l.finalizer; f != nil {
			f()
		}
		l = l.prev
	}
}

func (l *onSetDesktopHandlerList) handle(w *Window, dst **Desktop, src *Desktop) {
	if *dst == src {
		return
	}

	if l == nil {
		*dst = src
		return
	}

	w.beginUpdate()
	l.h(w, nil, dst, src)
	w.endUpdate()
}

func removeOnSetDesktopHandler(w *Window, l **onSetDesktopHandlerList) {
	node := *l
	*l = node.prev
	if f := node.finalizer; f != nil {
		f()
	}

}

// OnSetDurationHandler handles requests to change values of type time.Duration.
// If there was a previous handler installed, it's passed in prev. The handler
// then has the opportunity to call the previous handler before or after its
// own execution.
type OnSetDurationHandler func(w *Window, prev OnSetDurationHandler, dst *time.Duration, src time.Duration)

type onSetDurationHandlerList struct {
	prev      *onSetDurationHandlerList
	h         OnSetDurationHandler
	finalizer func()
}

func addOnSetDurationHandler(w *Window, l **onSetDurationHandlerList, h OnSetDurationHandler, finalizer func()) {
	prev := *l
	if prev == nil {
		*l = &onSetDurationHandlerList{
			h:         h,
			finalizer: finalizer,
		}
		return
	}

	*l = &onSetDurationHandlerList{
		prev: prev,
		h: func(_ *Window, _ OnSetDurationHandler, dst *time.Duration, src time.Duration) {
			h(w, prev.h, dst, src)
		},
		finalizer: finalizer,
	}
}

func (l *onSetDurationHandlerList) clear() {
	for l != nil {
		if f := l.finalizer; f != nil {
			f()
		}
		l = l.prev
	}
}

func (l *onSetDurationHandlerList) handle(w *Window, dst *time.Duration, src time.Duration) {
	if *dst == src {
		return
	}

	if l == nil {
		*dst = src
		return
	}

	w.beginUpdate()
	l.h(w, nil, dst, src)
	w.endUpdate()
}

func removeOnSetDurationHandler(w *Window, l **onSetDurationHandlerList) {
	node := *l
	*l = node.prev
	if f := node.finalizer; f != nil {
		f()
	}
}

// OnSetBoolHandler handles requests to change values of type bool.
// If there was a previous handler installed, it's passed in prev. The handler
// then has the opportunity to call the previous handler before or after its
// own execution.
type OnSetBoolHandler func(w *Window, prev OnSetBoolHandler, dst *bool, src bool)

type onSetBoolHandlerList struct {
	prev      *onSetBoolHandlerList
	h         OnSetBoolHandler
	finalizer func()
}

func addOnSetBoolHandler(w *Window, l **onSetBoolHandlerList, h OnSetBoolHandler, finalizer func()) {
	prev := *l
	if prev == nil {
		*l = &onSetBoolHandlerList{
			h:         h,
			finalizer: finalizer,
		}
		return
	}

	*l = &onSetBoolHandlerList{
		prev: prev,
		h: func(_ *Window, _ OnSetBoolHandler, dst *bool, src bool) {
			h(w, prev.h, dst, src)
		},
		finalizer: finalizer,
	}
}

func (l *onSetBoolHandlerList) clear() {
	for l != nil {
		if f := l.finalizer; f != nil {
			f()
		}
		l = l.prev
	}
}

func (l *onSetBoolHandlerList) handle(w *Window, dst *bool, src bool) {
	if *dst == src {
		return
	}

	if l == nil {
		*dst = src
		return
	}

	w.beginUpdate()
	l.h(w, nil, dst, src)
	w.endUpdate()
}

func removeOnSetBoolHandler(w *Window, l **onSetBoolHandlerList) {
	node := *l
	*l = node.prev
	if f := node.finalizer; f != nil {
		f()
	}
}

// OnSetIntHandler handles requests to change values of type int. If there
// was a previous handler installed, it's passed in prev. The handler then has
// the opportunity to call the previous handler before or after its own
// execution.
type OnSetIntHandler func(w *Window, prev OnSetIntHandler, dst *int, src int)

type onSetIntHandlerList struct {
	prev      *onSetIntHandlerList
	h         OnSetIntHandler
	finalizer func()
}

func addOnSetIntHandler(w *Window, l **onSetIntHandlerList, h OnSetIntHandler, finalizer func()) {
	prev := *l
	if prev == nil {
		*l = &onSetIntHandlerList{
			h:         h,
			finalizer: finalizer,
		}
		return
	}

	*l = &onSetIntHandlerList{
		prev: prev,
		h: func(_ *Window, _ OnSetIntHandler, dst *int, src int) {
			h(w, prev.h, dst, src)
		},
		finalizer: finalizer,
	}
}

func (l *onSetIntHandlerList) clear() {
	for l != nil {
		if f := l.finalizer; f != nil {
			f()
		}
		l = l.prev
	}
}

func (l *onSetIntHandlerList) handle(w *Window, dst *int, src int) {
	if *dst == src {
		return
	}

	if l == nil {
		*dst = src
		return
	}

	w.beginUpdate()
	l.h(w, nil, dst, src)
	w.endUpdate()
}

func removeOnSetIntHandler(w *Window, l **onSetIntHandlerList) {
	node := *l
	*l = node.prev
	if f := node.finalizer; f != nil {
		f()
	}
}

// OnSetPositionHandler handles requests to change values of type Position. If there
// was a previous handler installed, it's passed in prev. The handler then has
// the opportunity to call the previous handler before or after its own
// execution.
type OnSetPositionHandler func(w *Window, prev OnSetPositionHandler, dst *Position, src Position)

type onSetPositionHandlerList struct {
	prev      *onSetPositionHandlerList
	h         OnSetPositionHandler
	finalizer func()
}

func addOnSetPositionHandler(w *Window, l **onSetPositionHandlerList, h OnSetPositionHandler, finalizer func()) {
	prev := *l
	if prev == nil {
		*l = &onSetPositionHandlerList{
			h:         h,
			finalizer: finalizer,
		}
		return
	}

	*l = &onSetPositionHandlerList{
		prev: prev,
		h: func(_ *Window, _ OnSetPositionHandler, dst *Position, src Position) {
			h(w, prev.h, dst, src)
		},
		finalizer: finalizer,
	}
}

func (l *onSetPositionHandlerList) clear() {
	for l != nil {
		if f := l.finalizer; f != nil {
			f()
		}
		l = l.prev
	}
}

func (l *onSetPositionHandlerList) handle(w *Window, dst *Position, src Position) {
	if *dst == src {
		return
	}

	if l == nil {
		*dst = src
		return
	}

	w.beginUpdate()
	l.h(w, nil, dst, src)
	w.endUpdate()
}

func removeOnSetPositionHandler(w *Window, l **onSetPositionHandlerList) {
	node := *l
	*l = node.prev
	if f := node.finalizer; f != nil {
		f()
	}
}

// OnSetRectangleHandler handles requests to change values of type Rectangle.
// If there was a previous handler installed, it's passed in prev. The handler
// then has the opportunity to call the previous handler before or after its
// own execution.
type OnSetRectangleHandler func(w *Window, prev OnSetRectangleHandler, dst *Rectangle, src Rectangle)

type onSetRectangleHandlerList struct {
	prev      *onSetRectangleHandlerList
	h         OnSetRectangleHandler
	finalizer func()
}

func addOnSetRectangleHandler(w *Window, l **onSetRectangleHandlerList, h OnSetRectangleHandler, finalizer func()) {
	prev := *l
	if prev == nil {
		*l = &onSetRectangleHandlerList{
			h:         h,
			finalizer: finalizer,
		}
		return
	}

	*l = &onSetRectangleHandlerList{
		prev: prev,
		h: func(_ *Window, _ OnSetRectangleHandler, dst *Rectangle, src Rectangle) {
			h(w, prev.h, dst, src)
		},
		finalizer: finalizer,
	}
}

func (l *onSetRectangleHandlerList) clear() {
	for l != nil {
		if f := l.finalizer; f != nil {
			f()
		}
		l = l.prev
	}
}

func (l *onSetRectangleHandlerList) handle(w *Window, dst *Rectangle, src Rectangle) {
	if *dst == src {
		return
	}

	if l == nil {
		*dst = src
		return
	}

	w.beginUpdate()
	l.h(w, nil, dst, src)
	w.endUpdate()
}

func removeOnSetRectangleHandler(w *Window, l **onSetRectangleHandlerList) {
	node := *l
	*l = node.prev
	if f := node.finalizer; f != nil {
		f()
	}
}

// OnSetSizeHandler handles requests to change values of type Size. If there
// was a previous handler installed, it's passed in prev. The handler then has
// the opportunity to call the previous handler before or after its own
// execution.
type OnSetSizeHandler func(w *Window, prev OnSetSizeHandler, dst *Size, src Size)

type onSetSizeHandlerList struct {
	prev      *onSetSizeHandlerList
	h         OnSetSizeHandler
	finalizer func()
}

func addOnSetSizeHandler(w *Window, l **onSetSizeHandlerList, h OnSetSizeHandler, finalizer func()) {
	prev := *l
	if prev == nil {
		*l = &onSetSizeHandlerList{
			h:         h,
			finalizer: finalizer,
		}
		return
	}

	*l = &onSetSizeHandlerList{
		prev: prev,
		h: func(_ *Window, _ OnSetSizeHandler, dst *Size, src Size) {
			h(w, prev.h, dst, src)
		},
		finalizer: finalizer,
	}
}

func (l *onSetSizeHandlerList) clear() {
	for l != nil {
		if f := l.finalizer; f != nil {
			f()
		}
		l = l.prev
	}
}

func (l *onSetSizeHandlerList) handle(w *Window, dst *Size, src Size) {
	if *dst == src {
		return
	}

	if l == nil {
		*dst = src
		return
	}

	w.beginUpdate()
	l.h(w, nil, dst, src)
	w.endUpdate()
}

func removeOnSetSizeHandler(w *Window, l **onSetSizeHandlerList) {
	node := *l
	*l = node.prev
	if f := node.finalizer; f != nil {
		f()
	}
}

// OnSetStringHandler handles requests to change values of type String. If there
// was a previous handler installed, it's passed in prev. The handler then has
// the opportunity to call the previous handler before or after its own
// execution.
type OnSetStringHandler func(w *Window, prev OnSetStringHandler, dst *string, src string)

type onSetStringHandlerList struct {
	prev      *onSetStringHandlerList
	h         OnSetStringHandler
	finalizer func()
}

func addOnSetStringHandler(w *Window, l **onSetStringHandlerList, h OnSetStringHandler, finalizer func()) {
	prev := *l
	if prev == nil {
		*l = &onSetStringHandlerList{
			h:         h,
			finalizer: finalizer,
		}
		return
	}

	*l = &onSetStringHandlerList{
		prev: prev,
		h: func(_ *Window, _ OnSetStringHandler, dst *string, src string) {
			h(w, prev.h, dst, src)
		},
		finalizer: finalizer,
	}
}

func (l *onSetStringHandlerList) clear() {
	for l != nil {
		if f := l.finalizer; f != nil {
			f()
		}
		l = l.prev
	}
}

func (l *onSetStringHandlerList) handle(w *Window, dst *string, src string) {
	if *dst == src {
		return
	}

	if l == nil {
		*dst = src
		return
	}

	w.beginUpdate()
	l.h(w, nil, dst, src)
	w.endUpdate()
}

func removeOnSetStringHandler(w *Window, l **onSetStringHandlerList) {
	node := *l
	*l = node.prev
	if f := node.finalizer; f != nil {
		f()
	}
}

// OnSetStyleHandler handles requests to change values of type Style. If there
// was a previous handler installed, it's passed in prev. The handler then has
// the opportunity to call the previous handler before or after its own
// execution.
type OnSetStyleHandler func(w *Window, prev OnSetStyleHandler, dst *Style, src Style)

type onSetStyleHandlerList struct {
	prev      *onSetStyleHandlerList
	h         OnSetStyleHandler
	finalizer func()
}

func addOnSetStyleHandler(w *Window, l **onSetStyleHandlerList, h OnSetStyleHandler, finalizer func()) {
	prev := *l
	if prev == nil {
		*l = &onSetStyleHandlerList{
			h:         h,
			finalizer: finalizer,
		}
		return
	}

	*l = &onSetStyleHandlerList{
		prev: prev,
		h: func(_ *Window, _ OnSetStyleHandler, dst *Style, src Style) {
			h(w, prev.h, dst, src)
		},
		finalizer: finalizer,
	}
}

func (l *onSetStyleHandlerList) clear() {
	for l != nil {
		if f := l.finalizer; f != nil {
			f()
		}
		l = l.prev
	}
}

func (l *onSetStyleHandlerList) handle(w *Window, dst *Style, src Style) {
	if *dst == src {
		return
	}

	if l == nil {
		*dst = src
		return
	}

	w.beginUpdate()
	l.h(w, nil, dst, src)
	w.endUpdate()
}

func removeOnSetStyleHandler(w *Window, l **onSetStyleHandlerList) {
	node := *l
	*l = node.prev
	if f := node.finalizer; f != nil {
		f()
	}
}

// OnSetWindowHandler handles requests to change values of type *Window. If
// there was a previous handler installed, it's passed in prev. The handler
// then has the opportunity to call the previous handler before or after its
// own execution.
type OnSetWindowHandler func(w *Window, prev OnSetWindowHandler, dst **Window, src *Window)

type onSetWindowHandlerList struct {
	prev      *onSetWindowHandlerList
	h         OnSetWindowHandler
	finalizer func()
}

func addOnSetWindowHandler(w *Window, l **onSetWindowHandlerList, h OnSetWindowHandler, finalizer func()) {
	prev := *l
	if prev == nil {
		*l = &onSetWindowHandlerList{
			h:         h,
			finalizer: finalizer,
		}
		return
	}

	*l = &onSetWindowHandlerList{
		prev: prev,
		h: func(_ *Window, _ OnSetWindowHandler, dst **Window, src *Window) {
			h(w, prev.h, dst, src)
		},
		finalizer: finalizer,
	}
}

func (l *onSetWindowHandlerList) clear() {
	for l != nil {
		if f := l.finalizer; f != nil {
			f()
		}
		l = l.prev
	}
}

func (l *onSetWindowHandlerList) handle(w *Window, dst **Window, src *Window) {
	if *dst == src {
		return
	}

	if l == nil {
		*dst = src
		return
	}

	w.beginUpdate()
	l.h(w, nil, dst, src)
	w.endUpdate()
}

func removeOnSetWindowHandler(w *Window, l **onSetWindowHandlerList) {
	node := *l
	*l = node.prev
	if f := node.finalizer; f != nil {
		f()
	}

}

// OnSetWindowStyleHandler handles requests to change values of type
// WindowStyle. If there was a previous handler installed, it's passed in prev.
// The handler then has the opportunity to call the previous handler before or
// after its own execution.
type OnSetWindowStyleHandler func(w *Window, prev OnSetWindowStyleHandler, dst *WindowStyle, src WindowStyle)

type onSetWindowStyleHandlerList struct {
	prev      *onSetWindowStyleHandlerList
	h         OnSetWindowStyleHandler
	finalizer func()
}

func addOnSetWindowStyleHandler(w *Window, l **onSetWindowStyleHandlerList, h OnSetWindowStyleHandler, finalizer func()) {
	prev := *l
	if prev == nil {
		*l = &onSetWindowStyleHandlerList{
			h:         h,
			finalizer: finalizer,
		}
		return
	}

	*l = &onSetWindowStyleHandlerList{
		prev: prev,
		h: func(_ *Window, _ OnSetWindowStyleHandler, dst *WindowStyle, src WindowStyle) {
			h(w, prev.h, dst, src)
		},
		finalizer: finalizer,
	}
}

func (l *onSetWindowStyleHandlerList) clear() {
	for l != nil {
		if f := l.finalizer; f != nil {
			f()
		}
		l = l.prev
	}
}

func (l *onSetWindowStyleHandlerList) handle(w *Window, dst *WindowStyle, src WindowStyle) {
	if *dst == src {
		return
	}

	if l == nil {
		*dst = src
		return
	}

	w.beginUpdate()
	l.h(w, nil, dst, src)
	w.endUpdate()
}

func removeOnSetWindowStyleHandler(w *Window, l **onSetWindowStyleHandlerList) {
	node := *l
	*l = node.prev
	if f := node.finalizer; f != nil {
		f()
	}
}
