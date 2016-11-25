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
	view   Position
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

func addOnCloseHandler(l **onCloseHandlerList, h OnCloseHandler, finalizer func()) {
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
		h: func(w *Window, _ OnCloseHandler) {
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

func removeOnCloseHandler(l **onCloseHandlerList) {
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

func addOnKeyHandler(l **onKeyHandlerList, h OnKeyHandler, finalizer func()) {
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
		h: func(w *Window, _ OnKeyHandler, key tcell.Key, mod tcell.ModMask, r rune) bool {
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

func removeOnKeyHandler(l **onKeyHandlerList) {
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

func addOnMouseHandler(l **onMouseHandlerList, h OnMouseHandler, finalizer func()) {
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
		h: func(w *Window, _ OnMouseHandler, button tcell.ButtonMask, screenPos, winPos Position, mods tcell.ModMask) bool {
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

func removeOnMouseHandler(l **onMouseHandlerList) {
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

// OnPaintHandlerList represents a list of handlers subscribed to an event.
type OnPaintHandlerList struct {
	prev      *OnPaintHandlerList
	h         OnPaintHandler
	finalizer func()
}

// AddOnPaintHandler adds a handler to the handler list.
func AddOnPaintHandler(l **OnPaintHandlerList, h OnPaintHandler, finalizer func()) {
	prev := *l
	if prev == nil {
		*l = &OnPaintHandlerList{
			h: func(w *Window, _ OnPaintHandler, ctx PaintContext) {
				save := w.ctx
				w.ctx = ctx
				h(w, nil, ctx)
				w.ctx = save
			},
			finalizer: finalizer,
		}
		return
	}

	*l = &OnPaintHandlerList{
		prev: prev,
		h: func(w *Window, _ OnPaintHandler, ctx PaintContext) {
			save := w.ctx
			w.ctx = ctx
			h(w, prev.h, ctx)
			w.ctx = save
		},
		finalizer: finalizer,
	}
}

// Clear calls any finalizers on the handler list.
func (l *OnPaintHandlerList) Clear() {
	for l != nil {
		if f := l.finalizer; f != nil {
			f()
		}
		l = l.prev
	}
}

// Handle performs painting of ctx.
func (l *OnPaintHandlerList) Handle(w *Window, ctx PaintContext) {
	if l == nil || ctx.IsZero() {
		return
	}

	l.h(w, nil, ctx)
}

// RemoveOnPaintHandler undoes the most recent call to AddOnPaintHandler.
func RemoveOnPaintHandler(l **OnPaintHandlerList) {
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

func addOnSetDesktopHandler(l **onSetDesktopHandlerList, h OnSetDesktopHandler, finalizer func()) {
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
		h: func(w *Window, _ OnSetDesktopHandler, dst **Desktop, src *Desktop) {
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

func removeOnSetDesktopHandler(l **onSetDesktopHandlerList) {
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

func addOnSetDurationHandler(l **onSetDurationHandlerList, h OnSetDurationHandler, finalizer func()) {
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
		h: func(w *Window, _ OnSetDurationHandler, dst *time.Duration, src time.Duration) {
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

func removeOnSetDurationHandler(l **onSetDurationHandlerList) {
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

func addOnSetBoolHandler(l **onSetBoolHandlerList, h OnSetBoolHandler, finalizer func()) {
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
		h: func(w *Window, _ OnSetBoolHandler, dst *bool, src bool) {
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

func removeOnSetBoolHandler(l **onSetBoolHandlerList) {
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

// OnSetIntHandlerList represents a list of handlers subscribed to an event.
type OnSetIntHandlerList struct {
	prev      *OnSetIntHandlerList
	h         OnSetIntHandler
	finalizer func()
}

// AddOnSetIntHandler adds a handler to the handler list.
func AddOnSetIntHandler(l **OnSetIntHandlerList, h OnSetIntHandler, finalizer func()) {
	prev := *l
	if prev == nil {
		*l = &OnSetIntHandlerList{
			h:         h,
			finalizer: finalizer,
		}
		return
	}

	*l = &OnSetIntHandlerList{
		prev: prev,
		h: func(w *Window, _ OnSetIntHandler, dst *int, src int) {
			h(w, prev.h, dst, src)
		},
		finalizer: finalizer,
	}
}

// Clear calls any finalizers on the handler list.
func (l *OnSetIntHandlerList) Clear() {
	for l != nil {
		if f := l.finalizer; f != nil {
			f()
		}
		l = l.prev
	}
}

// Handle performs updating of dst from src or calling and associated handler.
func (l *OnSetIntHandlerList) Handle(w *Window, dst *int, src int) {
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

// RemoveOnSetIntHandler undoes the most recent call to AddOnSetIntHandler.
func RemoveOnSetIntHandler(l **OnSetIntHandlerList) {
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

// OnSetPositionHandlerList represents a list of handlers subscribed to an event.
type OnSetPositionHandlerList struct {
	prev      *OnSetPositionHandlerList
	h         OnSetPositionHandler
	finalizer func()
}

// AddOnSetPositionHandler adds a handler to the handler list.
func AddOnSetPositionHandler(l **OnSetPositionHandlerList, h OnSetPositionHandler, finalizer func()) {
	prev := *l
	if prev == nil {
		*l = &OnSetPositionHandlerList{
			h:         h,
			finalizer: finalizer,
		}
		return
	}

	*l = &OnSetPositionHandlerList{
		prev: prev,
		h: func(w *Window, _ OnSetPositionHandler, dst *Position, src Position) {
			h(w, prev.h, dst, src)
		},
		finalizer: finalizer,
	}
}

// Clear calls any finalizers on the handler list.
func (l *OnSetPositionHandlerList) Clear() {
	for l != nil {
		if f := l.finalizer; f != nil {
			f()
		}
		l = l.prev
	}
}

// Handle performs updating of dst from src or calling and associated handler.
func (l *OnSetPositionHandlerList) Handle(w *Window, dst *Position, src Position) {
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

// RemoveOnSetPositionHandler undoes the most recent call to
// AddOnSetPositionHandler.
func RemoveOnSetPositionHandler(l **OnSetPositionHandlerList) {
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

func addOnSetRectangleHandler(l **onSetRectangleHandlerList, h OnSetRectangleHandler, finalizer func()) {
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
		h: func(w *Window, _ OnSetRectangleHandler, dst *Rectangle, src Rectangle) {
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

func removeOnSetRectangleHandler(l **onSetRectangleHandlerList) {
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

// OnSetSizeHandlerList represents a list of handlers subscribed to an event.
type OnSetSizeHandlerList struct {
	prev      *OnSetSizeHandlerList
	h         OnSetSizeHandler
	finalizer func()
}

// AddOnSetSizeHandler adds a handler to the handler list.
func AddOnSetSizeHandler(l **OnSetSizeHandlerList, h OnSetSizeHandler, finalizer func()) {
	prev := *l
	if prev == nil {
		*l = &OnSetSizeHandlerList{
			h:         h,
			finalizer: finalizer,
		}
		return
	}

	*l = &OnSetSizeHandlerList{
		prev: prev,
		h: func(w *Window, _ OnSetSizeHandler, dst *Size, src Size) {
			h(w, prev.h, dst, src)
		},
		finalizer: finalizer,
	}
}

// Clear calls any finalizers on the handler list.
func (l *OnSetSizeHandlerList) Clear() {
	for l != nil {
		if f := l.finalizer; f != nil {
			f()
		}
		l = l.prev
	}
}

// Handle performs updating of dst from src or calling and associated handler.
func (l *OnSetSizeHandlerList) Handle(w *Window, dst *Size, src Size) {
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

// RemoveOnSetSizeHandler undoes the most recent call to AddOnSetSizeHandler.
func RemoveOnSetSizeHandler(l **OnSetSizeHandlerList) {
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

func addOnSetStringHandler(l **onSetStringHandlerList, h OnSetStringHandler, finalizer func()) {
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
		h: func(w *Window, _ OnSetStringHandler, dst *string, src string) {
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

func removeOnSetStringHandler(l **onSetStringHandlerList) {
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

// OnSetStyleHandlerList represents a list of handlers subscribed to an event.
type OnSetStyleHandlerList struct {
	prev      *OnSetStyleHandlerList
	h         OnSetStyleHandler
	finalizer func()
}

// AddOnSetStyleHandler adds a handler to the handler list.
func AddOnSetStyleHandler(l **OnSetStyleHandlerList, h OnSetStyleHandler, finalizer func()) {
	prev := *l
	if prev == nil {
		*l = &OnSetStyleHandlerList{
			h:         h,
			finalizer: finalizer,
		}
		return
	}

	*l = &OnSetStyleHandlerList{
		prev: prev,
		h: func(w *Window, _ OnSetStyleHandler, dst *Style, src Style) {
			h(w, prev.h, dst, src)
		},
		finalizer: finalizer,
	}
}

// Clear calls any finalizers on the handler list.
func (l *OnSetStyleHandlerList) Clear() {
	for l != nil {
		if f := l.finalizer; f != nil {
			f()
		}
		l = l.prev
	}
}

// Handle performs updating of dst from src or calling and associated handler.
func (l *OnSetStyleHandlerList) Handle(w *Window, dst *Style, src Style) {
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

// RemoveOnSetStyleHandler undoes the most recent call to AddOnSetStyleHandler.
func RemoveOnSetStyleHandler(l **OnSetStyleHandlerList) {
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

func addOnSetWindowHandler(l **onSetWindowHandlerList, h OnSetWindowHandler, finalizer func()) {
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
		h: func(w *Window, _ OnSetWindowHandler, dst **Window, src *Window) {
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

func removeOnSetWindowHandler(l **onSetWindowHandlerList) {
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

func addOnSetWindowStyleHandler(l **onSetWindowStyleHandlerList, h OnSetWindowStyleHandler, finalizer func()) {
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
		h: func(w *Window, _ OnSetWindowStyleHandler, dst *WindowStyle, src WindowStyle) {
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

func removeOnSetWindowStyleHandler(l **onSetWindowStyleHandlerList) {
	node := *l
	*l = node.prev
	if f := node.finalizer; f != nil {
		f()
	}
}
