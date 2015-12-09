// Copyright 2015 The WM Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wm

import (
	"fmt"
	"sync"

	"github.com/cznic/mathutil"
	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"
)

const (
	closeButtonOffset = 4 // X coordinate: Top border area width - closeButtonOffset
	closeButtonWidth  = 3
)

const (
	_ = iota
	dragPos
	dragRightSize
	dragLeftSize
	dragBottomSize
	dragULC
	dragURC
	dragLLC
	dragLRC
)

// Window represents a rectangular area of a screen. A window can have borders
// on all of its sizes.
//
// All exported methods of a Window are safe for concurrent access by multiple
// goroutines. However, concurrent calls of Window methods that mutate its
// state may produce flicker or corrupted screen rendering.
type Window struct {
	borderBotom          int                          //
	borderLeft           int                          //
	borderRight          int                          //
	borderTop            int                          //
	children             []*Window                    // In z-order.
	clientArea           Rectangle                    //
	closeButton          bool                         //
	desktop              *Desktop                     //
	dragScreenPos0       Position                     // Mouse screen position on drag event.
	dragState            int                          //
	dragWinPos0          Position                     // Window position on drag event.
	dragWinSize0         Size                         // Window size on drag event.
	focus                bool                         //
	focusedWindow        *Window                      // Root window only.
	mu                   sync.Mutex                   //
	onClearBorders       *onPaintHandlerList          //
	onClick              *onMouseHandlerList          //
	onClose              *onCloseHandlerList          //
	onDoubleClick        *onMouseHandlerList          //
	onDrag               *onMouseHandlerList          //
	onDrop               *onMouseHandlerList          //
	onKey                *onKeyHandlerList            //
	onMouseMove          *onMouseHandlerList          //
	onPaintBorderBottom  *onPaintHandlerList          //
	onPaintBorderLeft    *onPaintHandlerList          //
	onPaintBorderRight   *onPaintHandlerList          //
	onPaintBorderTop     *onPaintHandlerList          //
	onPaintChildren      *onPaintHandlerList          //
	onPaintClientArea    *onPaintHandlerList          //
	onPaintTitle         *onPaintHandlerList          //
	onSetBorderBotom     *onSetIntHandlerList         //
	onSetBorderLeft      *onSetIntHandlerList         //
	onSetBorderRight     *onSetIntHandlerList         //
	onSetBorderStyle     *onSetStyleHandlerList       //
	onSetBorderTop       *onSetIntHandlerList         //
	onSetClientAreaStyle *onSetStyleHandlerList       //
	onSetClientSize      *onSetSizeHandlerList        //
	onSetCloseButton     *onSetBoolHandlerList        //
	onSetFocus           *onSetBoolHandlerList        //
	onSetFocusedWindow   *onSetWindowHandlerList      // Root window only.
	onSetPosition        *onSetPositionHandlerList    //
	onSetSelection       *onSetRectangleHandlerList   // Root window only.
	onSetSize            *onSetSizeHandlerList        //
	onSetStyle           *onSetWindowStyleHandlerList //
	onSetTitle           *onSetStringHandlerList      //
	paintArea            Rectangle                    // Valid during painting.
	paintOrigin          Position                     // Valid during painting.
	parent               *Window                      //
	position             Position                     //
	selection            Rectangle                    // Root window only.
	size                 Size                         //
	style                WindowStyle                  //
	title                string                       //
}

func newWindow(desktop *Desktop, parent *Window, style WindowStyle) *Window {
	w := &Window{
		desktop: desktop,
		parent:  parent,
		style:   style,
	}
	addOnPaintHandler(w, &w.onClearBorders, w.onClearBordersHandler, nil)
	addOnPaintHandler(w, &w.onPaintChildren, w.onPaintChildrenHandler, nil)
	w.OnClick(w.onClickHandler, nil)
	w.OnDrag(w.onDragHandler, nil)
	w.OnDrop(w.onDropHandler, nil)
	w.OnMouseMove(w.onMouseMoveHandler, nil)
	w.OnPaintBorderBottom(w.onPaintBorderBottomHandler, nil)
	w.OnPaintBorderLeft(w.onPaintBorderLeftHandler, nil)
	w.OnPaintBorderRight(w.onPaintBorderRightHandler, nil)
	w.OnPaintBorderTop(w.onPaintBorderTopHandler, nil)
	w.OnPaintClientArea(w.onPaintClientAreaHandler, nil)
	w.OnPaintTitle(w.onPaintTitleHandler, nil)
	w.OnSetBorderBottom(w.onSetBorderBottomHandler, nil)
	w.OnSetBorderLeft(w.onSetBorderLeftHandler, nil)
	w.OnSetBorderRight(w.onSetBorderRightHandler, nil)
	w.OnSetBorderStyle(w.onSetBorderStyleHandler, nil)
	w.OnSetBorderTop(w.onSetBorderTopHandler, nil)
	w.OnSetClientAreaStyle(w.onSetClientAreaStyleHandler, nil)
	w.OnSetClientSize(w.onSetClientSizeHandler, nil)
	w.OnSetCloseButton(w.onSetCloseButtonHandler, nil)
	w.OnSetFocus(w.onSetFocusHandler, nil)
	w.OnSetPosition(w.onSetPositionHandler, nil)
	w.OnSetSize(w.onSetSizeHandler, nil)
	w.OnSetStyle(w.onSetStyleHandler, nil)
	w.OnSetTitle(w.onSetTitleHandler, nil)
	return w
}

func (w *Window) setCell(x, y int, mainc rune, combc []rune, style tcell.Style) {
	w.mu.Lock()
	o := w.position
	paintArea := w.paintArea
	paintOrigin := w.paintOrigin
	w.mu.Unlock()
	o.X += paintOrigin.X
	o.Y += paintOrigin.Y
	if !(Position{paintOrigin.X + x, paintOrigin.Y + y}).In(paintArea) {
		return
	}

	switch p := w.Parent(); p {
	case nil:
		app.setCell(o.X+x, o.Y+y, mainc, combc, style)
	default:
		p.setCell(o.X+x, o.Y+y, mainc, combc, style)
	}
}

func (w *Window) onPaintTitleHandler(_ *Window, prev OnPaintHandler, ctx PaintContext) {
	if prev != nil {
		panic("internal error")
	}

	title := w.Title()
	if title == "" {
		return
	}

	w.Printf(0, 0, w.Style().Title, " %s ", title)
}

func (w *Window) onSetTitleHandler(_ *Window, prev OnSetStringHandler, dst *string, src string) {
	if prev != nil {
		panic("internal error")
	}

	w.mu.Lock()
	*dst = src
	w.mu.Unlock()
	w.Invalidate(w.Area())
}

func (w *Window) onSetCloseButtonHandler(_ *Window, prev OnSetBoolHandler, dst *bool, src bool) {
	if prev != nil {
		panic("internal error")
	}

	w.mu.Lock()
	*dst = src
	w.mu.Unlock()
	w.Invalidate(w.Area())
}

func (w *Window) onClickHandler(_ *Window, prev OnMouseHandler, button tcell.ButtonMask, screenPos, pos Position, mods tcell.ModMask) bool {
	if prev != nil {
		panic("internal error")
	}

	clArea := w.ClientArea()
	if pos.In(clArea) {
		w.Lock()
		for i := len(w.children) - 1; i >= 0; i-- {
			ch := w.children[i]
			chPos := ch.position
			chPos.X += clArea.X
			chPos.Y += clArea.Y
			if r := (Rectangle{Position: chPos, Size: ch.size}); r.Has(pos) {
				w.Unlock()
				pos := pos
				pos.X -= chPos.X
				pos.Y -= chPos.Y
				ch.onClick.handle(ch, button, screenPos, pos, mods)
				return true
			}
		}
		w.Unlock()
	}
	w.BringToFront()
	w.SetFocus(true)
	if button != tcell.Button1 || mods != 0 {
		return false
	}

	if w.CloseButton() && pos.In(w.closeButtonArea()) {
		w.Close() //TODO CloseQuery
		return true
	}

	return !pos.In(clArea)
}

func (w *Window) onDragHandler(_ *Window, prev OnMouseHandler, button tcell.ButtonMask, screenPos, pos Position, mods tcell.ModMask) bool {
	if prev != nil {
		panic("internal error")
	}

	clArea := w.ClientArea()
	if pos.In(clArea) {
		w.Lock()
		for i := len(w.children) - 1; i >= 0; i-- {
			ch := w.children[i]
			chPos := ch.position
			chPos.X += clArea.X
			chPos.Y += clArea.Y
			if r := (Rectangle{Position: chPos, Size: ch.size}); r.Has(pos) {
				w.Unlock()
				pos := pos
				pos.X -= chPos.X
				pos.Y -= chPos.Y
				return ch.onDragHandler(nil, nil, button, screenPos, pos, mods)
			}
		}
		w.Unlock()
	}
	if button != tcell.Button1 || mods != 0 || w.Parent() == nil {
		return false
	}

	switch {
	case pos.In(w.topBorderDragMoveArea()):
		w.BringToFront()
		w.SetFocus(true)
		w.mu.Lock()
		w.dragState = dragPos
		w.dragScreenPos0 = screenPos
		w.dragWinPos0 = w.position
		w.mu.Unlock()
		return true
	case pos.In(w.rightBorderDragResizeArea()):
		w.BringToFront()
		w.SetFocus(true)
		w.mu.Lock()
		w.dragState = dragRightSize
		w.dragScreenPos0 = screenPos
		w.dragWinSize0 = w.size
		w.mu.Unlock()
		return true
	case pos.In(w.leftBorderDragResizeArea()):
		w.BringToFront()
		w.SetFocus(true)
		w.mu.Lock()
		w.dragState = dragLeftSize
		w.dragScreenPos0 = screenPos
		w.dragWinPos0 = w.position
		w.dragWinSize0 = w.size
		w.mu.Unlock()
		return true
	case pos.In(w.bottomBorderDragResizeArea()):
		w.BringToFront()
		w.SetFocus(true)
		w.mu.Lock()
		w.dragState = dragBottomSize
		w.dragScreenPos0 = screenPos
		w.dragWinSize0 = w.size
		w.mu.Unlock()
		return true
	case pos.In(w.borderLRCArea()):
		w.BringToFront()
		w.SetFocus(true)
		w.mu.Lock()
		w.dragState = dragLRC
		w.dragScreenPos0 = screenPos
		w.dragWinSize0 = w.size
		w.mu.Unlock()
		return true
	case pos.In(w.borderURCArea()):
		w.BringToFront()
		w.SetFocus(true)
		w.mu.Lock()
		w.dragState = dragURC
		w.dragScreenPos0 = screenPos
		w.dragWinPos0 = w.position
		w.dragWinSize0 = w.size
		w.mu.Unlock()
		return true
	case pos.In(w.borderLLCArea()):
		w.BringToFront()
		w.SetFocus(true)
		w.mu.Lock()
		w.dragState = dragLLC
		w.dragScreenPos0 = screenPos
		w.dragWinPos0 = w.position
		w.dragWinSize0 = w.size
		w.mu.Unlock()
		return true
	case pos.In(w.borderULCArea()):
		w.BringToFront()
		w.SetFocus(true)
		w.mu.Lock()
		w.dragState = dragULC
		w.dragScreenPos0 = screenPos
		w.dragWinPos0 = w.position
		w.dragWinSize0 = w.size
		w.mu.Unlock()
		return true
	default:
		return false
	}

}

func (w *Window) onMouseMoveHandler(_ *Window, prev OnMouseHandler, button tcell.ButtonMask, screenPos, pos Position, mods tcell.ModMask) bool {
	if prev != nil {
		panic("internal error")
	}

	if fw := w.Desktop().FocusedWindow(); fw != nil {
		fw.mu.Lock()
		ds := fw.dragState
		screenPos0 := fw.dragScreenPos0
		winPos0 := fw.dragWinPos0
		winSize0 := fw.dragWinSize0
		fw.mu.Unlock()
		dx := screenPos.X - screenPos0.X
		dy := screenPos.Y - screenPos0.Y

		switch ds {
		case dragPos:
			fw.SetPosition(Position{winPos0.X + dx, winPos0.Y + dy})
			return true
		case dragRightSize:
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width+dx), winSize0.Height})
			return true
		case dragLeftSize:
			if dx > winSize0.Width {
				dx = winSize0.Width - 1
			}
			fw.SetPosition(Position{winPos0.X + dx, winPos0.Y})
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width-dx), winSize0.Height})
			return true
		case dragBottomSize:
			fw.SetSize(Size{winSize0.Width, mathutil.Max(1, winSize0.Height+dy)})
			return true
		case dragLRC:
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width+dx), mathutil.Max(1, winSize0.Height+dy)})
			return true
		case dragURC:
			if dy > winSize0.Height {
				dy = winSize0.Height - 1
			}
			fw.SetPosition(Position{winPos0.X, winPos0.Y + dy})
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width+dx), mathutil.Max(1, winSize0.Height-dy)})
			return true
		case dragLLC:
			if dx > winSize0.Width {
				dx = winSize0.Width - 1
			}
			fw.SetPosition(Position{winPos0.X + dx, winPos0.Y})
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width-dx), mathutil.Max(1, winSize0.Height+dy)})
			return true
		case dragULC:
			if dx > winSize0.Width {
				dx = winSize0.Width - 1
			}
			if dy > winSize0.Height {
				dy = winSize0.Height - 1
			}
			fw.SetPosition(Position{winPos0.X + dx, winPos0.Y + dy})
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width-dx), mathutil.Max(1, winSize0.Height-dy)})
			return true
		}
	}

	clArea := w.ClientArea()
	if pos.In(clArea) {
		w.Lock()
		for i := len(w.children) - 1; i >= 0; i-- {
			ch := w.children[i]
			chPos := ch.position
			chPos.X += clArea.X
			chPos.Y += clArea.Y
			if r := (Rectangle{Position: chPos, Size: ch.size}); r.Has(pos) {
				w.Unlock()
				pos := pos
				pos.X -= chPos.X
				pos.Y -= chPos.Y
				return ch.onMouseMoveHandler(nil, nil, button, screenPos, pos, mods)
			}
		}
		w.mu.Unlock()
	}
	return false
}

func (w *Window) onDropHandler(_ *Window, prev OnMouseHandler, button tcell.ButtonMask, screenPos, pos Position, mods tcell.ModMask) bool {
	if prev != nil {
		panic("internal error")
	}

	if fw := w.Desktop().FocusedWindow(); fw != nil && button == tcell.Button1 && mods == 0 {
		fw.mu.Lock()
		ds := fw.dragState
		fw.dragState = 0
		screenPos0 := fw.dragScreenPos0
		winPos0 := fw.dragWinPos0
		winSize0 := fw.dragWinSize0
		fw.mu.Unlock()
		dx := screenPos.X - screenPos0.X
		dy := screenPos.Y - screenPos0.Y

		switch ds {
		case dragPos:
			fw.SetPosition(Position{winPos0.X + dx, winPos0.Y + dy})
			return true
		case dragRightSize:
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width+dx), winSize0.Height})
			return true
		case dragLeftSize:
			if dx > winSize0.Width {
				dx = winSize0.Width - 1
			}
			fw.SetPosition(Position{winPos0.X + dx, winPos0.Y})
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width-dx), winSize0.Height})
			return true
		case dragBottomSize:
			fw.SetSize(Size{winSize0.Width, mathutil.Max(1, winSize0.Height+dy)})
			return true
		case dragLRC:
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width+dx), mathutil.Max(1, winSize0.Height+dy)})
			return true
		case dragURC:
			if dy > winSize0.Height {
				dy = winSize0.Height - 1
			}
			fw.SetPosition(Position{winPos0.X, winPos0.Y + dy})
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width+dx), mathutil.Max(1, winSize0.Height-dy)})
			return true
		case dragLLC:
			if dx > winSize0.Width {
				dx = winSize0.Width - 1
			}
			fw.SetPosition(Position{winPos0.X + dx, winPos0.Y})
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width-dx), mathutil.Max(1, winSize0.Height+dy)})
			return true
		case dragULC:
			if dx > winSize0.Width {
				dx = winSize0.Width - 1
			}
			if dy > winSize0.Height {
				dy = winSize0.Height - 1
			}
			fw.SetPosition(Position{winPos0.X + dx, winPos0.Y + dy})
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width-dx), mathutil.Max(1, winSize0.Height-dy)})
			return true
		}
	}

	clPos := w.ClientPosition()
	w.Lock()
	for i := len(w.children) - 1; i >= 0; i-- {
		ch := w.children[i]
		chPos := ch.position
		chPos.X += clPos.X
		chPos.Y += clPos.Y
		if r := (Rectangle{Position: chPos, Size: ch.size}); r.Has(pos) {
			w.Unlock()
			pos := pos
			pos.X -= chPos.X
			pos.Y -= chPos.Y
			return ch.onDropHandler(nil, nil, button, screenPos, pos, mods)
		}
	}
	w.mu.Unlock()
	return false
}

func (w *Window) onClearBordersHandler(_ *Window, prev OnPaintHandler, ctx PaintContext) {
	if prev != nil {
		panic("internal error")
	}

	style := w.Style().Border.TCellStyle()
	if a := w.BorderTopArea(); a.Clip(ctx.Rectangle) {
		w.clear(a, style)
	}
	if a := w.BorderLeftArea(); a.Clip(ctx.Rectangle) {
		w.clear(a, style)
	}
	if a := w.BorderRightArea(); a.Clip(ctx.Rectangle) {
		w.clear(a, style)
	}
	if a := w.BorderBottomArea(); a.Clip(ctx.Rectangle) {
		w.clear(a, style)
	}
}

func (w *Window) onPaintBorderTopHandler(_ *Window, prev OnPaintHandler, ctx PaintContext) {
	if prev != nil {
		panic("internal error")
	}

	style := w.Style().Border
	tstyle := w.Style().Border.TCellStyle()
	sz := w.Size()
	borderArea := w.BorderTopArea()
	if borderArea.Width == 1 {
		w.SetCell(borderArea.X, borderArea.Y, ' ', nil, tstyle)
		return
	}

	for x := 0; x < borderArea.Width; x++ {
		var r rune
		switch x {
		case 0:
			r = tcell.RuneULCorner
			if sz.Height < 2 {
				r = ' '
			}
		case borderArea.Width - 1:
			r = tcell.RuneURCorner
			if sz.Height < 2 {
				r = ' '
			}
		default:
			r = tcell.RuneHLine
		}
		w.SetCell(x, 0, r, nil, tstyle)
	}

	if x := borderArea.Width - closeButtonOffset; x > 0 && w.CloseButton() {
		w.Printf(x, 0, style, "[X]")
	}
}

func (w *Window) onPaintBorderLeftHandler(_ *Window, prev OnPaintHandler, ctx PaintContext) {
	if prev != nil {
		panic("internal error")
	}

	style := w.Style().Border.TCellStyle()
	sz := w.Size()
	borderArea := w.BorderLeftArea()
	if borderArea.Height == 1 {
		w.SetCell(borderArea.X, borderArea.Y, ' ', nil, style)
		return
	}

	for y := 0; y < borderArea.Height; y++ {
		var r rune
		switch y {
		case 0:
			r = tcell.RuneULCorner
			if sz.Width < 2 {
				r = ' '
			}
		case borderArea.Height - 1:
			r = tcell.RuneLLCorner
			if sz.Width < 2 {
				r = ' '
			}
		default:
			r = tcell.RuneVLine
		}
		w.SetCell(0, y, r, nil, style)
	}
}

func (w *Window) onPaintBorderRightHandler(_ *Window, prev OnPaintHandler, ctx PaintContext) {
	if prev != nil {
		panic("internal error")
	}

	style := w.Style().Border.TCellStyle()
	sz := w.Size()
	borderArea := w.BorderRightArea()
	if borderArea.Height == 1 {
		w.SetCell(borderArea.X, borderArea.Y, ' ', nil, style)
		return
	}

	x := borderArea.Width - 1
	for y := 0; y < borderArea.Height; y++ {
		var r rune
		switch y {
		case 0:
			r = tcell.RuneURCorner
			if sz.Width < 2 {
				r = ' '
			}
		case borderArea.Height - 1:
			r = tcell.RuneLRCorner
			if sz.Width < 2 {
				r = ' '
			}
		default:
			r = tcell.RuneVLine
		}
		w.SetCell(x, y, r, nil, style)
	}
}

func (w *Window) onPaintBorderBottomHandler(_ *Window, prev OnPaintHandler, ctx PaintContext) {
	if prev != nil {
		panic("internal error")
	}

	style := w.Style().Border.TCellStyle()
	sz := w.Size()
	borderArea := w.BorderBottomArea()
	if borderArea.Width == 1 {
		w.SetCell(borderArea.X, borderArea.Y, ' ', nil, style)
		return
	}

	y := borderArea.Height - 1
	for x := 0; x < borderArea.Width; x++ {
		var r rune
		switch x {
		case 0:
			r = tcell.RuneLLCorner
			if sz.Height < 2 {
				r = ' '
			}
		case borderArea.Width - 1:
			r = tcell.RuneLRCorner
			if sz.Height < 2 {
				r = ' '
			}
		default:
			r = tcell.RuneHLine
		}
		w.SetCell(x, y, r, nil, style)
	}
}

func (w *Window) onSetSelectionHandler(_ *Window, prev OnSetRectangleHandler, dst *Rectangle, src Rectangle) {
	if prev != nil {
		panic("internal error")
	}

	app.beginUpdate()
	w.mu.Lock()
	*dst = src
	w.mu.Unlock()
	app.endUpdate()
}

func (w *Window) setFocusedWindow(v *Window) { w.onSetFocusedWindow.handle(w, &w.focusedWindow, v) }

func (w *Window) onSetFocusedWindowHandler(_ *Window, prev OnSetWindowHandler, dst **Window, src *Window) {
	if prev != nil {
		panic("internal error")
	}

	w.mu.Lock()
	old := *dst
	*dst = src
	w.mu.Unlock()
	if old != nil {
		old.SetFocus(false)
	}

	if src != nil {
		src.SetFocus(true)
	}

	if old != nil || src != nil {
		w.Invalidate(w.Area())
	}
}

func (w *Window) onSetFocusHandler(_ *Window, prev OnSetBoolHandler, dst *bool, src bool) {
	if prev != nil {
		panic("internal error")
	}

	w.mu.Lock()
	*dst = src
	d := w.desktop
	w.style.Border.Attr ^= tcell.AttrReverse
	w.mu.Unlock()

	switch {
	case src:
		d.SetFocusedWindow(w)
	default:
		d.SetFocusedWindow(nil)
	}
}

func (w *Window) onSetBorderStyleHandler(_ *Window, prev OnSetStyleHandler, dst *Style, src Style) {
	if prev != nil {
		panic("internal error")
	}

	w.mu.Lock()
	*dst = src
	w.mu.Unlock()
	w.Invalidate(w.BorderTopArea())
	w.Invalidate(w.BorderLeftArea())
	w.Invalidate(w.BorderRightArea())
	w.Invalidate(w.BorderBottomArea())
}

func (w *Window) onSetClientAreaStyleHandler(_ *Window, prev OnSetStyleHandler, dst *Style, src Style) {
	if prev != nil {
		panic("internal error")
	}

	w.mu.Lock()
	*dst = src
	w.mu.Unlock()
	w.InvalidateClientArea(w.ClientArea())
}

func (w *Window) onSetStyleHandler(_ *Window, prev OnSetWindowStyleHandler, dst *WindowStyle, src WindowStyle) {
	if prev != nil {
		panic("internal error")
	}

	w.mu.Lock()
	*dst = src
	w.mu.Unlock()
	w.Invalidate(w.Area())
}

func (w *Window) clear(area Rectangle, style tcell.Style) {
	for y := area.Y; y < area.Y+area.Height; y++ {
		for x := area.X; x < area.X+area.Width; x++ {
			w.SetCell(x, y, ' ', nil, style)
		}
	}
}

func (w *Window) onPaintClientAreaHandler(_ *Window, prev OnPaintHandler, ctx PaintContext) {
	if prev != nil {
		panic("internal error")
	}

	r := ctx.Rectangle
	r.X -= ctx.origin.X
	r.Y -= ctx.origin.Y
	w.clear(r, w.Style().ClientArea.TCellStyle())
}

func (w *Window) onPaintChildrenHandler(_ *Window, prev OnPaintHandler, ctx PaintContext) {
	if prev != nil {
		panic("internal error")
	}

	clPos := w.ClientPosition()
	for i := 0; ; i++ {
		c := w.Child(i)
		if c == nil {
			break
		}

		chPos := c.Position()
		chPos.X += clPos.X
		chPos.Y += clPos.Y
		if area := (Rectangle{chPos, c.Size()}); area.Clip(ctx.Rectangle) {
			area.X -= chPos.X
			area.Y -= chPos.Y
			c.paint(area)
		}
	}
}

func (w *Window) onSetPositionHandler(_ *Window, prev OnSetPositionHandler, dst *Position, src Position) {
	if prev != nil {
		panic("internal error")
	}

	w.mu.Lock()
	*dst = src
	p := w.parent
	w.mu.Unlock()
	p.InvalidateClientArea(p.ClientArea())
}

func (w *Window) onSetSizeHandler(_ *Window, prev OnSetSizeHandler, dst *Size, src Size) {
	if prev != nil {
		panic("internal error")
	}

	w.mu.Lock()
	src.Width = mathutil.Max(0, src.Width)
	src.Height = mathutil.Max(0, src.Height)
	*dst = src
	sz := Size{
		mathutil.Max(0, w.size.Width-(w.borderLeft+w.borderRight)),
		mathutil.Max(0, w.size.Height-(w.borderTop+w.borderBotom)),
	}
	p := w.parent
	w.mu.Unlock()

	w.SetClientSize(sz)
	if p != nil {
		p.InvalidateClientArea(p.ClientArea())
		return
	}

	w.Invalidate(w.Area())
}

func (w *Window) onSetClientSizeHandler(_ *Window, prev OnSetSizeHandler, dst *Size, src Size) {
	if prev != nil {
		panic("internal error")
	}

	w.mu.Lock()
	src.Width = mathutil.Min(mathutil.Max(0, src.Width), w.size.Width-(w.borderLeft+w.borderRight))
	src.Height = mathutil.Min(mathutil.Max(0, src.Height), w.size.Height-(w.borderTop+w.borderBotom))
	sz := Size{
		w.borderLeft + src.Width + w.borderRight,
		w.borderTop + src.Height + w.borderBotom,
	}
	p := w.parent
	w.mu.Unlock()

	w.SetSize(sz)
	if p != nil {
		p.InvalidateClientArea(p.ClientArea())
		return
	}

	w.InvalidateClientArea(w.ClientArea())
}

func (w *Window) onSetBorderBottomHandler(_ *Window, prev OnSetIntHandler, dst *int, src int) {
	if prev != nil {
		panic("internal error")
	}

	w.mu.Lock()
	*dst = src
	sz := Size{
		w.clientArea.Width,
		mathutil.Max(0, w.size.Height-(w.borderTop+w.borderBotom)),
	}
	w.mu.Unlock()

	w.SetClientSize(sz)
}

func (w *Window) onSetBorderLeftHandler(_ *Window, prev OnSetIntHandler, dst *int, src int) {
	if prev != nil {
		panic("internal error")
	}

	w.mu.Lock()
	*dst = src
	w.clientArea.X = src
	sz := Size{
		mathutil.Max(0, w.size.Width-(w.borderLeft+w.borderRight)),
		w.clientArea.Height,
	}
	w.mu.Unlock()

	w.SetClientSize(sz)
}

func (w *Window) onSetBorderRightHandler(_ *Window, prev OnSetIntHandler, dst *int, src int) {
	if prev != nil {
		panic("internal error")
	}

	w.mu.Lock()
	*dst = src
	sz := Size{
		mathutil.Max(0, w.size.Width-(w.borderLeft+w.borderRight)),
		w.clientArea.Height,
	}
	w.mu.Unlock()

	w.SetClientSize(sz)
}

func (w *Window) onSetBorderTopHandler(_ *Window, prev OnSetIntHandler, dst *int, src int) {
	if prev != nil {
		panic("internal error")
	}

	w.mu.Lock()
	*dst = src
	w.clientArea.Y = src
	sz := Size{
		w.clientArea.Width,
		mathutil.Max(0, w.size.Height-(w.borderTop+w.borderBotom)),
	}
	w.mu.Unlock()

	w.SetClientSize(sz)
}

func (w *Window) printCell(x, y, width int, main rune, comb []rune, style tcell.Style) (int, int) {
	switch main {
	case '\r':
		return 0, y
	case '\n':
		return 0, y + 1
	default:
		w.SetCell(x, y, main, comb, style)
		return x + width, y
	}
}

// beginUpdate marks the start of one or more updates to w.
func (w *Window) beginUpdate() {
	if w != nil {
		d := w.Desktop()
		d.mu.Lock()
		d.updateLevel++
		if d.updateLevel == 1 {
			d.invalidated = Rectangle{}
		}
		d.mu.Unlock()
		return
	}

	app.beginUpdate()
}

// endUpdate marks the end of one or more updates to w.
func (w *Window) endUpdate() {
	if w != nil {
		d := w.Desktop()
		d.mu.Lock()
		d.updateLevel--
		paint := d.updateLevel == 0
		invalidated := d.invalidated
		d.mu.Unlock()
		if paint && !invalidated.IsZero() {
			app.beginUpdate()
			d.Root().paint(invalidated)
			app.endUpdate()
		}
		return
	}

	app.endUpdate()
}

// setSize sets the window size.
func (w *Window) setSize(s Size) { w.onSetSize.handle(w, &w.size, s) }

func (w *Window) setPaintContext(area Rectangle, origin Position) (oldArea Rectangle, oldPosition Position) {
	w.mu.Lock()
	oldArea = w.paintArea
	oldPosition = w.paintOrigin
	w.paintArea = area
	w.paintOrigin = origin
	w.mu.Unlock()
	return oldArea, oldPosition
}

func (w *Window) click(button tcell.ButtonMask, pos Position, mods tcell.ModMask) {
	w.onClick.handle(w, button, pos, pos, mods)
}
func (w *Window) doubleClick(button tcell.ButtonMask, pos Position, mods tcell.ModMask) {
	w.onDoubleClick.handle(w, button, pos, pos, mods)
}
func (w *Window) drag(button tcell.ButtonMask, pos Position, mods tcell.ModMask) {
	w.onDrag.handle(w, button, pos, pos, mods)
}
func (w *Window) drop(button tcell.ButtonMask, pos Position, mods tcell.ModMask) {
	w.onDrop.handle(w, button, pos, pos, mods)
}
func (w *Window) mouseMove(pos Position, mods tcell.ModMask) {
	w.onMouseMove.handle(w, 0, pos, pos, mods)
}

// paint asks w to render an area.
func (w *Window) paint(area Rectangle) {
	d := w.Desktop()
	if area.IsZero() || !area.Clip(Rectangle{Size: getSize(w, &w.size)}) || d != app.Desktop() {
		return
	}

	d.mu.Lock()
	if d.updateLevel != 0 {
		for {
			p := w.Parent()
			if p == nil {
				d.invalidated.join(area)
				d.mu.Unlock()
				return
			}

			o := w.Position()
			cp := p.ClientPosition()
			area.X += o.X + cp.X
			area.Y += o.Y + cp.Y
			w = p
		}
	}
	d.mu.Unlock()

	a0 := w.Area()
	if a := a0; a.Clip(area) {
		w.onClearBorders.handle(w, PaintContext{a, a0.Position})
	}

	a0 = w.BorderTopArea()
	if a := a0; a.Clip(area) {
		w.onPaintBorderTop.handle(w, PaintContext{a, a0.Position})
	}

	if !a0.IsZero() && w.Title() != "" {
		a0.X++
		a0.Width--
		if w.CloseButton() {
			a0.Width -= closeButtonOffset
		}
		a0.Height = 1
		if a := a0; a.Clip(area) {
			w.onPaintTitle.handle(w, PaintContext{a, a0.Position})
		}
	}

	a0 = w.BorderLeftArea()
	if a := a0; a.Clip(area) {
		w.onPaintBorderLeft.handle(w, PaintContext{a, a0.Position})
	}

	a0 = w.ClientArea()
	if a := a0; a.Clip(area) {
		ctx := PaintContext{a, a0.Position}
		w.onPaintClientArea.handle(w, ctx)
		w.onPaintChildren.handle(w, ctx)
	}

	a0 = w.BorderRightArea()
	if a := a0; a.Clip(area) {
		w.onPaintBorderRight.handle(w, PaintContext{a, a0.Position})
	}

	a0 = w.BorderBottomArea()
	if a := a0; a.Clip(area) {
		w.onPaintBorderBottom.handle(w, PaintContext{a, a0.Position})
	}
}

func (w *Window) print(x, y int, style tcell.Style, s string) {
	if s == "" {
		return
	}

	if paintArea := getRectangle(w, &w.paintArea); paintArea.IsZero() { // Zero sized window or not in OnPaint.
		return
	}

	var main rune
	var comb []rune
	var state, width int

	const (
		st0         = iota // main, width, comb not valid.
		stCheckComb        // main, width, comb valid, checking if followed by combining char(s).
		stComb             // main, width, comb valid, collecting combining chars.
	)

	for _, r := range s {
		if r == 0 {
			continue
		}

		switch runewidth.RuneWidth(r) {
		case 0: // Combining char.
			switch state {
			case st0:
				main = ' '
				width = 1
				comb = append(comb[:0], r)
				state = stComb
			case stCheckComb:
				comb = append(comb, r)
				state = stComb
			case stComb:
				comb = append(comb, r)
				state = stComb
			default:
				panic("internal error")
			}
		case 1: // Normal width.
			switch state {
			case st0:
				main = r
				width = 1
				comb = comb[:0]
				state = stCheckComb
			case stCheckComb:
				x, y = w.printCell(x, y, width, main, comb, style)
				main = r
				width = 1
				comb = comb[:0]
				state = stCheckComb
			case stComb:
				comb = append(comb, r)
				state = stComb
			default:
				panic("internal error")
			}
		case 2: // Double width.
			switch state {
			case st0:
				main = r
				width = 2
				comb = comb[:0]
				state = stCheckComb
			case stCheckComb:
				x, y = w.printCell(x, y, width, main, comb, style)
				main = r
				width = 2
				comb = comb[:0]
				state = stCheckComb
			case stComb:
				comb = append(comb, r)
				state = stComb
			default:
				panic("internal error")
			}
		default:
			panic("internal error")
		}
	}
	switch state {
	case stCheckComb, stComb:
		w.printCell(x, y, width, main, comb, style)
	default:
		panic(fmt.Errorf("%q: %v", s, state))
	}
}

func (w *Window) bringToFront(c *Window) {
	if w == nil {
		return
	}

	if p := w.Parent(); p != nil {
		p.bringToFront(w)
	}
	w.mu.Lock()
	for i, v := range w.children {
		if v == c {
			if i == len(w.children)-1 { // Already in front.
				w.mu.Unlock()
				return
			}

			copy(w.children[i:], w.children[i+1:])
			w.children[len(w.children)-1] = c
			break
		}
	}
	w.mu.Unlock()
	w.InvalidateClientArea(w.ClientArea())
}

func (w *Window) removeChild(ch *Window) {
	w.mu.Lock()
	for i, v := range w.children {
		if v == ch {
			copy(w.children[i:], w.children[i+1:])
			w.children = w.children[:len(w.children)-1]
			break
		}
	}
	w.mu.Unlock()
}

func (w *Window) closeButtonArea() (r Rectangle) {
	w.mu.Lock()
	r.X = w.size.Width - closeButtonOffset
	r.Width = closeButtonWidth
	r.Height = 1
	w.mu.Unlock()
	return r
}

func (w *Window) topBorderDragMoveArea() (r Rectangle) {
	r = w.BorderTopArea()
	r.X++
	r.Width -= 2
	r.Height = 1
	return r
}

func (w *Window) leftBorderDragResizeArea() (r Rectangle) {
	r = w.BorderLeftArea()
	r.Y++
	r.Height -= 2
	r.Width = 1
	return r
}

func (w *Window) rightBorderDragResizeArea() (r Rectangle) {
	r = w.BorderRightArea()
	r.Y++
	r.Height -= 2
	r.X += r.Width - 1
	return r
}

func (w *Window) bottomBorderDragResizeArea() (r Rectangle) {
	r = w.BorderBottomArea()
	r.X++
	r.Width -= 2
	r.Y -= r.Height - 1
	r.Height = 1
	return r
}

func (w *Window) borderLRCArea() (r Rectangle) {
	r = w.BorderRightArea()
	r.X += r.Width - 1
	r.Y += r.Height - 1
	r.Width = 1
	r.Height = 1
	return r
}

func (w *Window) borderURCArea() (r Rectangle) {
	r = w.BorderRightArea()
	r.X += r.Width - 1
	r.Width = 1
	r.Height = 1
	return r
}

func (w *Window) borderLLCArea() (r Rectangle) {
	r = w.BorderBottomArea()
	r.Y += r.Height - 1
	r.Width = 1
	r.Height = 1
	return r
}

func (w *Window) borderULCArea() (r Rectangle) {
	r = w.BorderTopArea()
	r.Width = 1
	r.Height = 1
	return r
}

// ----------------------------------------------------------------------------

// Area returns the area of the window.
func (w *Window) Area() Rectangle { return Rectangle{Size: getSize(w, &w.size)} }

// BorderBottom returns the height of the bottom border.
func (w *Window) BorderBottom() int { return getInt(w, &w.borderBotom) }

// BorderBottomArea returns the area of the bottom border.
func (w *Window) BorderBottomArea() (r Rectangle) {
	w.mu.Lock()
	r.Y = w.size.Height - w.borderBotom
	r.Width = w.size.Width
	r.Height = w.borderBotom
	w.mu.Unlock()
	return r
}

// BorderLeft returns the width of the left border.
func (w *Window) BorderLeft() int { return getInt(w, &w.borderLeft) }

// BorderLeftArea returns the area of the left border.
func (w *Window) BorderLeftArea() (r Rectangle) {
	w.mu.Lock()
	r.Width = w.borderLeft
	r.Height = w.size.Height
	w.mu.Unlock()
	return r
}

// BorderRight returns the width of the right border.
func (w *Window) BorderRight() int { return getInt(w, &w.borderRight) }

// BorderRightArea returns the area of the right border.
func (w *Window) BorderRightArea() (r Rectangle) {
	w.mu.Lock()
	r.X = w.size.Width - w.borderRight
	r.Width = w.borderRight
	r.Height = w.size.Height
	w.mu.Unlock()
	return r
}

// BorderStyle returns the border style.
func (w *Window) BorderStyle() Style { return getStyle(w, &w.style.Border) }

// BorderTop returns the height of the top border.
func (w *Window) BorderTop() int { return getInt(w, &w.borderTop) }

// BorderTopArea returns the area of the top border.
func (w *Window) BorderTopArea() (r Rectangle) {
	w.mu.Lock()
	r.Width = w.size.Width
	r.Height = w.borderTop
	w.mu.Unlock()
	return r
}

// BringToFront puts a child window on top of all its siblings. The method has
// no effect if w is a root window or if w is docked.
func (w *Window) BringToFront() { w.Parent().bringToFront(w) }

// Child returns the nth child window or nil if no such exists.
func (w *Window) Child(n int) (r *Window) {
	w.mu.Lock()
	if n < len(w.children) {
		r = w.children[n]
	}
	w.mu.Unlock()
	return r
}

// Children returns the number of child windows.
func (w *Window) Children() (r int) {
	w.mu.Lock()
	r = len(w.children)
	w.mu.Unlock()
	return r
}

// ClientArea returns the client area.
func (w *Window) ClientArea() Rectangle { return getRectangle(w, &w.clientArea) }

// ClientPosition returns the position of the client area relative to w.
func (w *Window) ClientPosition() Position { return getPosition(w, &w.clientArea.Position) }

// ClientSize returns the size of the client area.
func (w *Window) ClientSize() Size { return getSize(w, &w.clientArea.Size) }

// ClientAreaStyle returns the client area style.
func (w *Window) ClientAreaStyle() Style { return getStyle(w, &w.style.ClientArea) }

// Close closes w.
func (w *Window) Close() {
	w.onClose.handle(w)
	w.SetFocus(false)
	if p := w.Parent(); p != nil {
		p.removeChild(w)
		p.InvalidateClientArea(p.ClientArea())
	}

	w.onClearBorders.clear()
	w.onClick.clear()
	w.onClose.clear()
	w.onDoubleClick.clear()
	w.onDrag.clear()
	w.onDrop.clear()
	w.onKey.clear()
	w.onMouseMove.clear()
	w.onPaintBorderBottom.clear()
	w.onPaintBorderLeft.clear()
	w.onPaintBorderRight.clear()
	w.onPaintBorderTop.clear()
	w.onPaintChildren.clear()
	w.onPaintClientArea.clear()
	w.onPaintTitle.clear()
	w.onSetBorderBotom.clear()
	w.onSetBorderLeft.clear()
	w.onSetBorderRight.clear()
	w.onSetBorderStyle.clear()
	w.onSetBorderTop.clear()
	w.onSetClientAreaStyle.clear()
	w.onSetClientSize.clear()
	w.onSetCloseButton.clear()
	w.onSetFocus.clear()
	w.onSetFocusedWindow.clear()
	w.onSetPosition.clear()
	w.onSetSelection.clear()
	w.onSetSize.clear()
	w.onSetStyle.clear()
	w.onSetTitle.clear()
}

// CloseButton returns whether the window shows a close button.
func (w *Window) CloseButton() bool { return getBool(w, &w.closeButton) }

// Desktop returns which Desktop w appears on.
func (w *Window) Desktop() *Desktop { return getDesktop(w, &w.desktop) }

// Focus returns wheter the window is focused.
func (w *Window) Focus() bool { return getBool(w, &w.focus) }

// Invalidate marks a window area for repaint.
func (w *Window) Invalidate(area Rectangle) {
	if !area.Clip(Rectangle{Size: getSize(w, &w.size)}) {
		return
	}

	w.beginUpdate()
	p := w.Parent()
	if p == nil {
		w.paint(area)
		w.endUpdate()
		return
	}

	o := w.Position()
	cp := p.ClientPosition()
	area.X += o.X + cp.X
	area.Y += o.Y + cp.Y
	p.paint(area)
	w.endUpdate()
}

// InvalidateClientArea marks an area of the client area for repaint.
func (w *Window) InvalidateClientArea(area Rectangle) {
	if !area.Clip(getRectangle(w, &w.clientArea)) {
		return
	}

	w.beginUpdate()
	p := w.Parent()
	if p == nil {
		w.paint(area)
		w.endUpdate()
		return
	}

	o := w.Position()
	cp := p.ClientPosition()
	area.X += o.X + cp.X
	area.Y += o.Y + cp.Y
	p.paint(area)
	w.endUpdate()
}

// Lock will acquire an exclusive lock of w or of the Application instance if w
// is nil. Lock is intended only for the various OnSet* handlers to protect the
// statement when setting a particular property value. Example:
//
//	func onSetFocus(w *wm.Window, prev wm.OnSetBoolHandler, dst *bool, src bool) {
//		if prev != nil {
//			prev(w, nil, dst, src)
//		} else {
//			w.Lock()
//			*dst = src
//			w.Unlock()
//		}
//	}
func (w *Window) Lock() {
	if w != nil {
		w.mu.Lock()
		return
	}

	app.mu.Lock()
}

// NewChild creates a child window.
func (w *Window) NewChild(area Rectangle) *Window {
	c := newWindow(getDesktop(w, &w.desktop), w, app.ChildWindowStyle())
	w.mu.Lock()
	w.children = append(w.children, c)
	w.mu.Unlock()
	c.SetBorderTop(1)
	c.SetBorderLeft(1)
	c.SetBorderRight(1)
	c.SetBorderBottom(1)
	c.SetPosition(area.Position)
	c.SetSize(area.Size)
	return c
}

// OnClick sets a mouse click event handler. When the event handler is removed,
// finalize is called, if not nil.
func (w *Window) OnClick(h OnMouseHandler, finalize func()) {
	addOnMouseHandler(w, &w.onClick, h, finalize)
}

// OnClose sets a window close event handler. When the event handler is
// removed, finalize is called, if not nil.
func (w *Window) OnClose(h OnCloseHandler, finalize func()) {
	addOnCloseHandler(w, &w.onClose, h, finalize)
}

// OnDoubleClick sets a mouse double click event handler. When the event
// handler is removed, finalize is called, if not nil.
func (w *Window) OnDoubleClick(h OnMouseHandler, finalize func()) {
	addOnMouseHandler(w, &w.onDoubleClick, h, finalize)
}

// OnDrag sets a mouse drag event handler. When the event handler is removed,
// finalize is called, if not nil.
func (w *Window) OnDrag(h OnMouseHandler, finalize func()) {
	addOnMouseHandler(w, &w.onDrag, h, finalize)
}

// OnDrop sets a mouse drop event handler. When the event handler is removed,
// finalize is called, if not nil.
func (w *Window) OnDrop(h OnMouseHandler, finalize func()) {
	addOnMouseHandler(w, &w.onDrop, h, finalize)
}

// OnKey sets a key event handler. When the event handler is removed, finalize
// is called, if not nil.
func (w *Window) OnKey(h OnKeyHandler, finalize func()) {
	addOnKeyHandler(w, &w.onKey, h, finalize)
}

// OnMouseMove sets a mouse move event handler. When the event handler is
// removed, finalize is called, if not nil.
func (w *Window) OnMouseMove(h OnMouseHandler, finalize func()) {
	addOnMouseHandler(w, &w.onMouseMove, h, finalize)
}

// OnPaintClientArea sets a client area paint handler. When the event handler
// is removed, finalize is called, if not nil. Example:
//
//	func onPaintClientArea(w *wm.Window, prev wm.OnPaintHandler, area wm.Rectangle) {
//		if prev != nil {
//			prev(w, nil, area)
//		}
//		w.Printf(0, 0, w.Style(), "Hello 世界!\nTime: %s", time.Now())
//	}
//
//	...
//
//	w.OnPaintClientArea(onPaintClientArea, nil)
func (w *Window) OnPaintClientArea(h OnPaintHandler, finalize func()) {
	addOnPaintHandler(w, &w.onPaintClientArea, h, finalize)
}

// OnPaintBorderBottom sets a bottom border paint handler. When the event
// handler is removed, finalize is called, if not nil. Example:
//
//	func onPaintBorderBottom(w *wm.Window, prev wm.OnPaintHandler, area wm.Rectangle) {
//		if prev != nil {
//			prev(w, nil, area)
//		}
//		style := w.Style().TCellStyle()
//		w := w.BorderBottomArea().Width
//		for x := 0; x < w; x++ {
//			var r rune
//			switch x {
//			case 0:
//				r = tcell.RuneLLCorner
//			case w - 1:
//				r = tcell.RuneLRCorner
//			default:
//				r = tcell.RuneHLine
//			}
//			w.SetCell(x, 0, r, nil, style)
//		}
//	}
//
//	...
//
//	w.OnPaintBorderBottom(onPaintBorderBottom, nil)
//	w.SetBorderBottom(1)
func (w *Window) OnPaintBorderBottom(h OnPaintHandler, finalize func()) {
	addOnPaintHandler(w, &w.onPaintBorderBottom, h, finalize)
}

// OnPaintBorderLeft sets a left border paint handler. When the event handler
// is removed, finalize is called, if not nil. Example:
//
//	func onPaintBorderLeft(w *wm.Window, prev wm.OnPaintHandler, area wm.Rectangle) {
//		if prev != nil {
//			prev(w, nil, area)
//		}
//		style := w.Style().TCellStyle()
//		h := w.BorderLeftArea().Height
//		for y := 0; y < h; y++ {
//			var r rune
//			switch y {
//			case 0:
//				r = tcell.RuneULCorner
//			case h - 1:
//				r = tcell.RuneLLCorner
//			default:
//				r = tcell.RuneVLine
//			}
//			w.SetCell(0, y, r, nil, style)
//		}
//	}
//
//	...
//
//	w.OnPaintBorderLeft(onPaintBorderLeft, nil)
//	w.SetBorderLeft(1)
func (w *Window) OnPaintBorderLeft(h OnPaintHandler, finalize func()) {
	addOnPaintHandler(w, &w.onPaintBorderLeft, h, finalize)
}

// OnPaintBorderRight sets a right border paint handler. When the event handler
// is removed, finalize is called, if not nil. Example:
//
//	func onPaintBorderRight(w *wm.Window, prev wm.OnPaintHandler, area wm.Rectangle) {
//		if prev != nil {
//			prev(w, nil, area)
//		}
//		style := w.Style().TCellStyle()
//		h := w.BorderRightArea().Height
//		for y := 0; y < h; y++ {
//			var r rune
//			switch y {
//			case 0:
//				r = tcell.RuneURCorner
//			case h - 1:
//				r = tcell.RuneLRCorner
//			default:
//				r = tcell.RuneVLine
//			}
//			w.SetCell(0, y, r, nil, style)
//		}
//	}
//
//	...
//
//	w.OnPaintBorderRight(onPaintBorderRight, nil)
//	w.SetBorderRight(1)
func (w *Window) OnPaintBorderRight(h OnPaintHandler, finalize func()) {
	addOnPaintHandler(w, &w.onPaintBorderRight, h, finalize)
}

// OnPaintBorderTop sets a top border paint handler. When the event handler
// is removed, finalize is called, if not nil. Example:
//
//	func onPaintBorderTop(w *wm.Window, prev wm.OnPaintHandler, area wm.Rectangle) {
//		if prev != nil {
//			prev(w, nil, area)
//		}
//		style := w.Style().TCellStyle()
//		w := w.BorderTopArea().Width
//		for x := 0; x < w; x++ {
//			var r rune
//			switch x {
//			case 0:
//				r = tcell.RuneULCorner
//			case w - 1:
//				r = tcell.RuneURCorner
//			default:
//				r = tcell.RuneHLine
//			}
//			w.SetCell(x, 0, r, nil, style)
//		}
//	}
//
//	...
//
//	w.OnPaintBorderTop(onPaintBorderTop, nil)
//	w.SetBorderTop(1)
func (w *Window) OnPaintBorderTop(h OnPaintHandler, finalize func()) {
	addOnPaintHandler(w, &w.onPaintBorderTop, h, finalize)
}

// OnPaintTitle sets a window title paint handler. When the event handler is
// removed, finalize is called, if not nil. Example:
func (w *Window) OnPaintTitle(h OnPaintHandler, finalize func()) {
	addOnPaintHandler(w, &w.onPaintTitle, h, finalize)
}

// OnSetBorderBottom sets a handler invoked on SetBorderBottom. When the event
// handler is removed, finalize is called, if not nil.
func (w *Window) OnSetBorderBottom(h OnSetIntHandler, finalize func()) {
	addOnSetIntHandler(w, &w.onSetBorderBotom, h, finalize)
}

// OnSetBorderLeft sets a handler invoked on SetBorderLeft. When the event
// handler is removed, finalize is called, if not nil.
func (w *Window) OnSetBorderLeft(h OnSetIntHandler, finalize func()) {
	addOnSetIntHandler(w, &w.onSetBorderLeft, h, finalize)
}

// OnSetBorderRight sets a handler invoked on SetBorderRight. When the event
// handler is removed, finalize is called, if not nil.
func (w *Window) OnSetBorderRight(h OnSetIntHandler, finalize func()) {
	addOnSetIntHandler(w, &w.onSetBorderRight, h, finalize)
}

// OnSetBorderStyle sets a handler invoked on SetBorderStyle. When the event
// handler is removed, finalize is called, if not nil.
func (w *Window) OnSetBorderStyle(h OnSetStyleHandler, finalize func()) {
	addOnSetStyleHandler(w, &w.onSetBorderStyle, h, finalize)
}

// OnSetBorderTop sets a handler invoked on SetBorderTop. When the event
// handler is removed, finalize is called, if not nil.
func (w *Window) OnSetBorderTop(h OnSetIntHandler, finalize func()) {
	addOnSetIntHandler(w, &w.onSetBorderTop, h, finalize)
}

// OnSetClientAreaStyle sets a handler invoked on SetClientAreaStyle. When the
// event handler is removed, finalize is called, if not nil.
func (w *Window) OnSetClientAreaStyle(h OnSetStyleHandler, finalize func()) {
	addOnSetStyleHandler(w, &w.onSetClientAreaStyle, h, finalize)
}

// OnSetClientSize sets a handler invoked on SetClientSize. When the event
// handler is removed, finalize is called, if not nil.
func (w *Window) OnSetClientSize(h OnSetSizeHandler, finalize func()) {
	addOnSetSizeHandler(w, &w.onSetClientSize, h, finalize)
}

// OnSetCloseButton sets a handler invoked on SetCloseButton. When the event
// handler is removed, finalize is called, if not nil.
func (w *Window) OnSetCloseButton(h OnSetBoolHandler, finalize func()) {
	addOnSetBoolHandler(w, &w.onSetCloseButton, h, finalize)
}

// OnSetFocus sets a handler invoked on SetFocus. When the event handler is
// removed, finalize is called, if not nil.
func (w *Window) OnSetFocus(h OnSetBoolHandler, finalize func()) {
	addOnSetBoolHandler(w, &w.onSetFocus, h, finalize)
}

// OnSetPosition sets a handler invoked on SetPosition. When the event handler
// is removed, finalize is called, if not nil.
func (w *Window) OnSetPosition(h OnSetPositionHandler, finalize func()) {
	addOnSetPositionHandler(w, &w.onSetPosition, h, finalize)
}

// OnSetSize sets a handler invoked on SetSize. When the event handler is
// removed, finalize is called, if not nil.
func (w *Window) OnSetSize(h OnSetSizeHandler, finalize func()) {
	addOnSetSizeHandler(w, &w.onSetSize, h, finalize)
}

// OnSetStyle sets a handler invoked on SetStyle. When the event handler is
// removed, finalize is called, if not nil.
func (w *Window) OnSetStyle(h OnSetWindowStyleHandler, finalize func()) {
	addOnSetWindowStyleHandler(w, &w.onSetStyle, h, finalize)
}

// OnSetTitle sets a handler invoked on SetTitle. When the event handler is
// removed, finalize is called, if not nil.
func (w *Window) OnSetTitle(h OnSetStringHandler, finalize func()) {
	addOnSetStringHandler(w, &w.onSetTitle, h, finalize)
}

// Printf prints format with arguments at x, y. Calling this method outside of
// an OnPaint handler is ignored.
//
// Special handling:
//
//	'\r'	x, y = 0, y
//	'\n'	x, y = 0, y+1
func (w *Window) Printf(x, y int, style Style, format string, arg ...interface{}) {
	if paintArea := getRectangle(w, &w.paintArea); paintArea.IsZero() { // Zero sized window or not in OnPaint.
		return
	}

	w.print(x, y, style.TCellStyle(), fmt.Sprintf(format, arg...))
}

// Parent returns the window's parent. Root windows have nil parent.
func (w *Window) Parent() *Window { return getWindow(w, &w.parent) }

// Position returns the window position relative to its parent.
func (w *Window) Position() Position { return getPosition(w, &w.position) }

// RemoveOnClick undoes the most recent OnClick call. The function will panic if
// there is no handler set.
func (w *Window) RemoveOnClick() { removeOnMouseHandler(w, &w.onClick) }

// RemoveOnClose undoes the most recent OnClose call. The function will panic
// if there is no handler set.
func (w *Window) RemoveOnClose() { removeOnCloseHandler(w, &w.onClose) }

// RemoveOnDoubleClick undoes the most recent OnDoubleClick call. The function
// will panic if there is no handler set.
func (w *Window) RemoveOnDoubleClick() { removeOnMouseHandler(w, &w.onDoubleClick) }

// RemoveOnDrag undoes the most recent OnDrag call. The function will panic if
// there is no handler set.
func (w *Window) RemoveOnDrag() { removeOnMouseHandler(w, &w.onDrag) }

// RemoveOnDrop undoes the most recent OnDrop call. The function will panic if
// there is no handler set.
func (w *Window) RemoveOnDrop() { removeOnMouseHandler(w, &w.onDrop) }

// RemoveOnKey undoes the most recent OnKey call. The function will panic if
// there is no handler set.
func (w *Window) RemoveOnKey() { removeOnKeyHandler(w, &w.onKey) }

// RemoveOnMouseMove undoes the most recent OnMouseMove call. The function will
// panic if there is no handler set.
func (w *Window) RemoveOnMouseMove() { removeOnMouseHandler(w, &w.onMouseMove) }

// RemoveOnPaintClientArea undoes the most recent OnPaintClientArea call. The
// function will panic if there is no handler set.
func (w *Window) RemoveOnPaintClientArea() { removeOnPaintHandler(w, &w.onPaintClientArea) }

// RemoveOnPaintBorderBottom undoes the most recent OnPaintBorderBottom call.
// The function will panic if there is no handler set.
func (w *Window) RemoveOnPaintBorderBottom() { removeOnPaintHandler(w, &w.onPaintBorderBottom) }

// RemoveOnPaintBorderLeft undoes the most recent OnPaintBorderLeft call.
// The function will panic if there is no handler set.
func (w *Window) RemoveOnPaintBorderLeft() { removeOnPaintHandler(w, &w.onPaintBorderLeft) }

// RemoveOnPaintBorderRight undoes the most recent OnPaintBorderRight call.
// The function will panic if there is no handler set.
func (w *Window) RemoveOnPaintBorderRight() { removeOnPaintHandler(w, &w.onPaintBorderRight) }

// RemoveOnPaintBorderTop undoes the most recent OnPaintBorderTop call.
// The function will panic if there is no handler set.
func (w *Window) RemoveOnPaintBorderTop() { removeOnPaintHandler(w, &w.onPaintBorderTop) }

// RemoveOnPaintTitle undoes the most recent OnPaintTitle call.  The function
// will panic if there is no handler set.
func (w *Window) RemoveOnPaintTitle() { removeOnPaintHandler(w, &w.onPaintTitle) }

// RemoveOnSetBorderBottom undoes the most recent OnSetBorderBottom call. The
// function will panic if there is no handler set.
func (w *Window) RemoveOnSetBorderBottom() { removeOnSetIntHandler(w, &w.onSetBorderBotom) }

// RemoveOnSetBorderLeft undoes the most recent OnSetBorderLeft call. The
// function will panic if there is no handler set.
func (w *Window) RemoveOnSetBorderLeft() { removeOnSetIntHandler(w, &w.onSetBorderLeft) }

// RemoveOnSetBorderRight undoes the most recent OnSetBorderRight call. The
// function will panic if there is no handler set.
func (w *Window) RemoveOnSetBorderRight() { removeOnSetIntHandler(w, &w.onSetBorderRight) }

// RemoveOnSetBorderStyle undoes the most recent OnSetBorderStyle call. The
// function will panic if there is no handler set.
func (w *Window) RemoveOnSetBorderStyle() { removeOnSetStyleHandler(w, &w.onSetBorderStyle) }

// RemoveOnSetBorderTop undoes the most recent OnSetBorderTop call. The
// function will panic if there is no handler set.
func (w *Window) RemoveOnSetBorderTop() { removeOnSetIntHandler(w, &w.onSetBorderTop) }

// RemoveOnSetClientAreaStyle undoes the most recent OnSetClientAreaStyle call.
// The function will panic if there is no handler set.
func (w *Window) RemoveOnSetClientAreaStyle() { removeOnSetStyleHandler(w, &w.onSetClientAreaStyle) }

// RemoveOnSetClientSize undoes the most recent OnSetClientSize call. The
// function will panic if there is no handler set.
func (w *Window) RemoveOnSetClientSize() { removeOnSetSizeHandler(w, &w.onSetClientSize) }

// RemoveOnSetCloseButton undoes the most recent OnSetCloseButton call. The
// function will panic if there is no handler set.
func (w *Window) RemoveOnSetCloseButton() { removeOnSetBoolHandler(w, &w.onSetCloseButton) }

// RemoveOnSetFocus undoes the most recent OnSetFocus call. The function will
// panic if there is no handler set.
func (w *Window) RemoveOnSetFocus() { removeOnSetBoolHandler(w, &w.onSetFocus) }

// RemoveOnSetPosition undoes the most recent OnSetPosition call. The function
// will panic if there is no handler set.
func (w *Window) RemoveOnSetPosition() { removeOnSetPositionHandler(w, &w.onSetPosition) }

// RemoveOnSetSize undoes the most recent OnSetSize call. The function will
// panic if there is no handler set.
func (w *Window) RemoveOnSetSize() { removeOnSetSizeHandler(w, &w.onSetSize) }

// RemoveOnSetStyle undoes the most recent OnSetStyle call. The function will
// panic if there is no handler set.
func (w *Window) RemoveOnSetStyle() { removeOnSetWindowStyleHandler(w, &w.onSetStyle) }

// RemoveOnSetTitle undoes the most recent OnSetTitle call. The function will
// panic if there is no handler set.
func (w *Window) RemoveOnSetTitle() { removeOnSetStringHandler(w, &w.onSetTitle) }

// SetBorderBottom sets the height of the bottom border.
func (w *Window) SetBorderBottom(v int) { w.onSetBorderBotom.handle(w, &w.borderBotom, v) }

// SetBorderLeft sets the width of the left border.
func (w *Window) SetBorderLeft(v int) { w.onSetBorderLeft.handle(w, &w.borderLeft, v) }

// SetBorderRight sets the width of the right border.
func (w *Window) SetBorderRight(v int) { w.onSetBorderRight.handle(w, &w.borderRight, v) }

// SetBorderStyle sets the border style.
func (w *Window) SetBorderStyle(s Style) { w.onSetBorderStyle.handle(w, &w.style.Border, s) }

// SetBorderTop sets the height of the top border.
func (w *Window) SetBorderTop(v int) { w.onSetBorderTop.handle(w, &w.borderTop, v) }

// SetCell renders a single character cell. Calling this method outside of an
// OnPaint* handler is ignored.
func (w *Window) SetCell(x, y int, mainc rune, combc []rune, style tcell.Style) {
	w.beginUpdate()
	w.setCell(x, y, mainc, combc, style)
	w.endUpdate()
}

// SetClientAreaStyle sets the client area style.
func (w *Window) SetClientAreaStyle(s Style) { w.onSetClientAreaStyle.handle(w, &w.style.ClientArea, s) }

// SetClientSize sets the size of the client area.
func (w *Window) SetClientSize(s Size) { w.onSetSize.handle(w, &w.clientArea.Size, s) }

// SetCloseButton sets whether the window shows a close button.
func (w *Window) SetCloseButton(v bool) {
	if getWindow(w, &w.parent) != nil {
		w.onSetCloseButton.handle(w, &w.closeButton, v)
	}
}

// SetFocus sets whether the window is focused.
func (w *Window) SetFocus(v bool) { w.onSetFocus.handle(w, &w.focus, v) }

// SetPosition sets the window position relative to its parent.
func (w *Window) SetPosition(p Position) {
	if getWindow(w, &w.parent) != nil {
		w.onSetPosition.handle(w, &w.position, p)
	}
}

// SetSize sets the window size.
func (w *Window) SetSize(s Size) {
	if getWindow(w, &w.parent) != nil {
		w.setSize(s)
	}
}

// SetStyle sets the window style.
func (w *Window) SetStyle(s WindowStyle) { w.onSetStyle.handle(w, &w.style, s) }

// SetTitle sets the window title.
func (w *Window) SetTitle(s string) { w.onSetTitle.handle(w, &w.title, s) }

// Size returns the window size.
func (w *Window) Size() Size { return getSize(w, &w.size) }

// Style returns the window style.
func (w *Window) Style() WindowStyle { return getWindowStyle(w, &w.style) }

// Title returns the window title.
func (w *Window) Title() string { return getString(w, &w.title) }

// Unlock will unlock w.
func (w *Window) Unlock() {
	if w != nil {
		w.mu.Unlock()
		return
	}

	app.mu.Unlock()
}
