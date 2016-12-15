// Copyright 2015 The WM Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wm

import (
	"fmt"
	"time"

	"github.com/cznic/mathutil"
	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"
)

const (
	closeButtonOffset = 4 // X coordinate: Top border area width - closeButtonOffset
	closeButtonWidth  = 3
)

const (
	_ = iota //TODOOK
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
// on all of its sides and a title.
//
// Window methods must be called only directly from an event handler goroutine
// or from a function that was enqueued using Application.Post or
// Application.PostWait.
type Window struct {
	borderBottom         int                          // Height.
	borderLeft           int                          // Width.
	borderRight          int                          // Width.
	borderTop            int                          // Height.
	children             []*Window                    // In z-order.
	clientArea           Rectangle                    // In window coordinates, excludes any borders.
	closeButton          bool                         // Enable.
	ctx                  PaintContext                 // Valid during painting.
	desktop              *Desktop                     // Which Desktop this window belongs to. Never changes.
	dragScreenPos0       Position                     // Mouse screen position on drag event.
	dragState            int                          // One of the drag{Pos,RightSize,...} constants,
	dragWinPos0          Position                     // Window position on drag event.
	dragWinSize0         Size                         // Window size on drag event.
	dragWindow           *Window                      // Which window will receive mouse move and drop events.
	dragWindowPos        Position                     // In parent window coordinates.
	focus                bool                         // Whether this window has focus.
	focusedWindow        *Window                      // Root window only.
	onClearBorders       *OnPaintHandlerList          //
	onClearClientArea    *OnPaintHandlerList          //
	onClick              *OnMouseHandlerList          //
	onClickBorder        *OnMouseHandlerList          //
	onClose              *onCloseHandlerList          //
	onDoubleClick        *OnMouseHandlerList          //
	onDoubleClickBorder  *OnMouseHandlerList          //
	onDrag               *OnMouseHandlerList          //
	onDragBorder         *OnMouseHandlerList          //
	onDrop               *OnMouseHandlerList          //
	onKey                *onKeyHandlerList            //
	onMouseMove          *OnMouseHandlerList          //
	onPaintBorderBottom  *OnPaintHandlerList          //
	onPaintBorderLeft    *OnPaintHandlerList          //
	onPaintBorderRight   *OnPaintHandlerList          //
	onPaintBorderTop     *OnPaintHandlerList          //
	onPaintChildren      *OnPaintHandlerList          //
	onPaintClientArea    *OnPaintHandlerList          //
	onPaintTitle         *OnPaintHandlerList          //
	onSetBorderBotom     *OnSetIntHandlerList         //
	onSetBorderLeft      *OnSetIntHandlerList         //
	onSetBorderRight     *OnSetIntHandlerList         //
	onSetBorderStyle     *OnSetStyleHandlerList       //
	onSetBorderTop       *OnSetIntHandlerList         //
	onSetClientAreaStyle *OnSetStyleHandlerList       //
	onSetClientSize      *OnSetSizeHandlerList        //
	onSetCloseButton     *OnSetBoolHandlerList        //
	onSetFocus           *OnSetBoolHandlerList        //
	onSetFocusedWindow   *onSetWindowHandlerList      // Root window only.
	onSetOrigin          *OnSetPositionHandlerList    //
	onSetPosition        *OnSetPositionHandlerList    //
	onSetSelection       *onSetRectangleHandlerList   // Root window only.
	onSetSize            *OnSetSizeHandlerList        //
	onSetStyle           *onSetWindowStyleHandlerList //
	onSetTitle           *onSetStringHandlerList      //
	parent               *Window                      // Nil for root window.
	position             Position                     // In parent window coordinates.
	rendered             time.Duration                //
	selection            Rectangle                    // Root window only.
	size                 Size                         //
	style                WindowStyle                  //
	title                string                       //
	view                 Position                     // Viewport origin.
}

func newWindow(desktop *Desktop, parent *Window, style WindowStyle) *Window {
	w := &Window{
		desktop: desktop,
		parent:  parent,
		style:   style,
	}
	AddOnPaintHandler(&w.onClearBorders, w.onClearBordersHandler, nil)
	AddOnPaintHandler(&w.onClearClientArea, w.onClearClientAreaHandler, nil)
	AddOnPaintHandler(&w.onPaintChildren, w.onPaintChildrenHandler, nil)
	w.OnClickBorder(w.onClickBorderHandler, nil)
	w.OnDragBorder(w.onDragBorderHandler, nil)
	w.OnPaintBorderBottom(w.onPaintBorderBottomHandler, nil)
	w.OnPaintBorderLeft(w.onPaintBorderLeftHandler, nil)
	w.OnPaintBorderRight(w.onPaintBorderRightHandler, nil)
	w.OnPaintBorderTop(w.onPaintBorderTopHandler, nil)
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
	w.OnSetOrigin(w.onSetOriginHandler, nil)
	w.OnSetPosition(w.onSetPositionHandler, nil)
	w.OnSetSize(w.onSetSizeHandler, nil)
	w.OnSetStyle(w.onSetStyleHandler, nil)
	w.OnSetTitle(w.onSetTitleHandler, nil)
	return w
}

func (w *Window) setCell(p Position, mainc rune, combc []rune, style tcell.Style) {
	if !w.ctx.origin.add(p).In(w.ctx.Rectangle) {
		return
	}

	p = p.add(w.position).add(w.ctx.origin).sub(w.ctx.view)
	switch w := w.Parent(); w {
	case nil:
		App.setCell(p, mainc, combc, style)
	default:
		w.setCell(p, mainc, combc, style)
	}
}

func (w *Window) onPaintTitleHandler(_ *Window, prev OnPaintHandler, _ PaintContext) {
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

	*dst = src
	w.Invalidate(w.BorderTopArea())
}

func (w *Window) onSetCloseButtonHandler(_ *Window, prev OnSetBoolHandler, dst *bool, src bool) {
	if prev != nil {
		panic("internal error")
	}

	*dst = src
	w.Invalidate(w.BorderTopArea())
}

func (w *Window) onClickBorderHandler(_ *Window, prev OnMouseHandler, button tcell.ButtonMask, screenPos, pos Position, mods tcell.ModMask) bool {
	if button != tcell.Button1 || mods != 0 {
		return false
	}

	w.BringToFront()
	w.SetFocus(true)
	if w.CloseButton() && pos.In(w.closeButtonArea()) {
		w.Close() //TODO CloseQuery
		return true
	}

	return false

}

func (w *Window) onDragBorderHandler(_ *Window, prev OnMouseHandler, button tcell.ButtonMask, screenPos, pos Position, mods tcell.ModMask) bool {
	if prev != nil {
		panic("internal error")
	}

	if button != tcell.Button1 || mods != 0 || w.Parent() == nil {
		return false
	}

	switch {
	case pos.In(w.topBorderDragMoveArea()):
		w.BringToFront()
		w.SetFocus(true)
		w.dragState = dragPos
		w.dragScreenPos0 = screenPos
		w.dragWinPos0 = w.position
		return true
	case pos.In(w.rightBorderDragResizeArea()):
		w.BringToFront()
		w.SetFocus(true)
		w.dragState = dragRightSize
		w.dragScreenPos0 = screenPos
		w.dragWinSize0 = w.size
		return true
	case pos.In(w.leftBorderDragResizeArea()):
		w.BringToFront()
		w.SetFocus(true)
		w.dragState = dragLeftSize
		w.dragScreenPos0 = screenPos
		w.dragWinPos0 = w.position
		w.dragWinSize0 = w.size
		return true
	case pos.In(w.bottomBorderDragResizeArea()):
		w.BringToFront()
		w.SetFocus(true)
		w.dragState = dragBottomSize
		w.dragScreenPos0 = screenPos
		w.dragWinSize0 = w.size
		return true
	case pos.In(w.borderLRCArea()):
		w.BringToFront()
		w.SetFocus(true)
		w.dragState = dragLRC
		w.dragScreenPos0 = screenPos
		w.dragWinSize0 = w.size
		return true
	case pos.In(w.borderURCArea()):
		w.BringToFront()
		w.SetFocus(true)
		w.dragState = dragURC
		w.dragScreenPos0 = screenPos
		w.dragWinPos0 = w.position
		w.dragWinSize0 = w.size
		return true
	case pos.In(w.borderLLCArea()):
		w.BringToFront()
		w.SetFocus(true)
		w.dragState = dragLLC
		w.dragScreenPos0 = screenPos
		w.dragWinPos0 = w.position
		w.dragWinSize0 = w.size
		return true
	case pos.In(w.borderULCArea()):
		w.BringToFront()
		w.SetFocus(true)
		w.dragState = dragULC
		w.dragScreenPos0 = screenPos
		w.dragWinPos0 = w.position
		w.dragWinSize0 = w.size
		return true
	default:
		return false
	}

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

func (w *Window) onPaintBorderTopHandler(_ *Window, prev OnPaintHandler, _ PaintContext) {
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

func (w *Window) onPaintBorderLeftHandler(_ *Window, prev OnPaintHandler, _ PaintContext) {
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

func (w *Window) onPaintBorderRightHandler(_ *Window, prev OnPaintHandler, _ PaintContext) {
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

func (w *Window) onPaintBorderBottomHandler(_ *Window, prev OnPaintHandler, _ PaintContext) {
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

	App.BeginUpdate()
	*dst = src
	App.EndUpdate()
}

func (w *Window) setFocusedWindow(u *Window) { w.onSetFocusedWindow.handle(w, &w.focusedWindow, u) }

// ATM root window only.
func (w *Window) onSetFocusedWindowHandler(_ *Window, prev OnSetWindowHandler, dst **Window, src *Window) {
	if prev != nil || w != w.Desktop().Root() {
		panic("internal error")
	}

	old := *dst
	*dst = src
	if old != nil {
		old.SetFocus(false)
		if old.Parent() != nil {
			old.Invalidate(old.Area())
		}
	}

	if src != nil {
		src.SetFocus(true)
		if src.Parent() != nil {
			src.Invalidate(src.Area())
		}
	}
}

func (w *Window) onSetFocusHandler(_ *Window, prev OnSetBoolHandler, dst *bool, src bool) {
	if prev != nil {
		panic("internal error")
	}

	*dst = src
	d := w.desktop
	w.style.Border.Attr ^= tcell.AttrReverse

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

	*dst = src
	w.Invalidate(w.BorderTopArea())
	w.Invalidate(w.BorderLeftArea())
	w.Invalidate(w.BorderRightArea())
	w.Invalidate(w.BorderBottomArea())
}

func (w *Window) onSetClientAreaStyleHandler(_ *Window, prev OnSetStyleHandler, dst *Style, src Style) {
	if prev != nil {
		panic("internal error")
	}

	*dst = src
	w.InvalidateClientArea(w.ClientArea())
}

func (w *Window) onSetStyleHandler(_ *Window, prev OnSetWindowStyleHandler, dst *WindowStyle, src WindowStyle) {
	if prev != nil {
		panic("internal error")
	}

	*dst = src
	w.Invalidate(w.Area())
}

func (w *Window) clear(area Rectangle, style tcell.Style) {
	for y := area.Y; y < area.Y+area.Height; y++ {
		for x := area.X; x < area.X+area.Width; x++ {
			w.SetCell(x, y, ' ', nil, style)
		}
	}
}

func (w *Window) onClearClientAreaHandler(_ *Window, prev OnPaintHandler, ctx PaintContext) {
	if prev != nil {
		panic("internal error")
	}

	w.clear(Rectangle{ctx.sub(ctx.origin), ctx.Rectangle.Size}, w.Style().ClientArea.TCellStyle())
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

		chPos := c.Position().add(clPos)
		if area := (Rectangle{chPos, c.Size()}); area.Clip(ctx.Rectangle) {
			c.paint(Rectangle{area.sub(chPos), area.Size})
		}
	}
}

func (w *Window) onSetOriginHandler(_ *Window, prev OnSetPositionHandler, dst *Position, src Position) {
	if prev != nil {
		panic("internal error")
	}

	w.Invalidate(w.ClientArea())
	*dst = src
}

func (w *Window) onSetPositionHandler(_ *Window, prev OnSetPositionHandler, dst *Position, src Position) {
	if prev != nil {
		panic("internal error")
	}

	w.Invalidate(w.Area())
	*dst = src
	w.Invalidate(w.Area())
}

func (w *Window) onSetSizeHandler(_ *Window, prev OnSetSizeHandler, dst *Size, src Size) {
	if prev != nil {
		panic("internal error")
	}

	src.Width = mathutil.Max(0, src.Width)
	src.Height = mathutil.Max(0, src.Height)
	w.Invalidate(w.Area())
	*dst = src
	csz := Size{
		mathutil.Max(0, src.Width-(w.borderLeft+w.borderRight)),
		mathutil.Max(0, src.Height-(w.borderTop+w.borderBottom)),
	}
	w.SetClientSize(csz)
	w.Invalidate(w.Area())
}

func (w *Window) onSetClientSizeHandler(_ *Window, prev OnSetSizeHandler, dst *Size, src Size) {
	if prev != nil {
		panic("internal error")
	}

	src.Width = mathutil.Max(0, src.Width)
	src.Height = mathutil.Max(0, src.Height)
	w.Invalidate(w.Area())
	*dst = src
	wsz := Size{
		w.borderLeft + src.Width + w.borderRight,
		w.borderTop + src.Height + w.borderBottom,
	}
	p := w.parent

	w.SetSize(wsz)
	if p != nil {
		w.Invalidate(w.Area())
		return
	}

	w.InvalidateClientArea(w.ClientArea())
}

func (w *Window) onSetBorderBottomHandler(_ *Window, prev OnSetIntHandler, dst *int, src int) {
	if prev != nil {
		panic("internal error")
	}

	*dst = src
	sz := Size{w.clientArea.Width, mathutil.Max(0, w.size.Height-(w.borderTop+w.borderBottom))}
	w.SetClientSize(sz)
}

func (w *Window) onSetBorderLeftHandler(_ *Window, prev OnSetIntHandler, dst *int, src int) {
	if prev != nil {
		panic("internal error")
	}

	*dst = src
	w.clientArea.X = src
	sz := Size{mathutil.Max(0, w.size.Width-(w.borderLeft+w.borderRight)), w.clientArea.Height}
	w.SetClientSize(sz)
}

func (w *Window) onSetBorderRightHandler(_ *Window, prev OnSetIntHandler, dst *int, src int) {
	if prev != nil {
		panic("internal error")
	}

	*dst = src
	sz := Size{mathutil.Max(0, w.size.Width-(w.borderLeft+w.borderRight)), w.clientArea.Height}
	w.SetClientSize(sz)
}

func (w *Window) onSetBorderTopHandler(_ *Window, prev OnSetIntHandler, dst *int, src int) {
	if prev != nil {
		panic("internal error")
	}

	*dst = src
	w.clientArea.Y = src
	sz := Size{w.clientArea.Width, mathutil.Max(0, w.size.Height-(w.borderTop+w.borderBottom))}
	w.SetClientSize(sz)
}

func (w *Window) printCell(x, y, width int, main rune, comb []rune, style tcell.Style) (int, int) {
	switch main {
	case '\t':
		return x + 8 - x%8, y
	case '\n':
		return 0, y + 1
	case '\r':
		return 0, y
	default:
		w.SetCell(x, y, main, comb, style)
		return x + width, y
	}
}

// BeginUpdate marks the start of one or more updates to w.
//
// Failing to properly pair BeginUpdate with a corresponding EndUpdate will
// cause application screen corruption and/or freeze.
func (w *Window) BeginUpdate() {
	if w != nil {
		d := w.Desktop()
		d.updateLevel++
		if d.updateLevel == 1 {
			d.invalidated = Rectangle{}
		}
		return
	}

	App.BeginUpdate()
}

// EndUpdate marks the end of one or more updates to w.
//
// Failing to properly pair BeginUpdate with a corresponding EndUpdate will
// cause application screen corruption and/or freeze.
func (w *Window) EndUpdate() {
	if w != nil {
		d := w.Desktop()
		d.updateLevel--
		invalidated := d.invalidated
		if d.updateLevel == 0 && !invalidated.IsZero() {
			App.BeginUpdate()
			r := d.Root()
			t := time.Now()
			r.paint(invalidated)
			r.rendered = time.Since(t)
			App.EndUpdate()
		}
		return
	}

	App.EndUpdate()
}

// setSize sets the window size.
func (w *Window) setSize(s Size) { w.onSetSize.Handle(w, &w.size, s) }

func (w *Window) findEventTarget(winPos Position, clientAreaHandler, borderHandler func(*Window, Position)) (*Window, Position, func(*Window, Position)) {
search:
	winPos2 := winPos.add(w.view)
	clArea := w.ClientArea()
	if winPos.In(clArea) {
		winPos = winPos2.sub(clArea.Position)
		var chArea Rectangle
		for i := len(w.children) - 1; i >= 0; i-- {
			ch := w.children[i]
			chArea = ch.Area()
			chArea.Position = ch.Position()
			if winPos.In(chArea) {
				winPos = winPos.sub(chArea.Position)
				w = ch
				goto search
			}
		}

		return w, winPos, clientAreaHandler
	}

	return w, winPos, borderHandler
}

func (w *Window) event(winPos Position, clientAreaHandler, borderHandler func(*Window, Position), setFocus bool) {
	w, pos, handler := w.findEventTarget(winPos, clientAreaHandler, borderHandler)
	if setFocus {
		w.BringToFront()
		w.SetFocus(true)
	}
	handler(w, pos)
}

func (w *Window) click(button tcell.ButtonMask, screenPos Position, mods tcell.ModMask) {
	w.event(
		screenPos,
		func(w *Window, winPos Position) {
			w.onClick.Handle(w, button, screenPos, winPos, mods)
		},
		func(w *Window, winPos Position) {
			w.onClickBorder.Handle(w, button, screenPos, winPos, mods)
		},
		true,
	)
}

func (w *Window) doubleClick(button tcell.ButtonMask, screenPos Position, mods tcell.ModMask) {
	w.event(
		screenPos,
		func(w *Window, winPos Position) {
			w.onDoubleClick.Handle(w, button, screenPos, winPos, mods)
		},
		func(w *Window, winPos Position) {
			w.onDoubleClickBorder.Handle(w, button, screenPos, winPos, mods)
		},
		true,
	)
}

func (w *Window) drag(button tcell.ButtonMask, screenPos Position, mods tcell.ModMask) {
	w.dragWindow = nil
	w.event(
		screenPos,
		func(cw *Window, winPos Position) {
			if cw.onDrag.Handle(cw, button, screenPos, winPos, mods) {
				w.dragWindow = cw
				w.dragWindowPos = winPos
			}
		},
		func(cw *Window, winPos Position) {
			if cw.onDragBorder.Handle(cw, button, screenPos, winPos, mods) {
				w.dragWindow = cw
				w.dragWindowPos = winPos
			}
		},
		true,
	)
}
func (w *Window) drop(button tcell.ButtonMask, screenPos Position, mods tcell.ModMask) {
	defer func() { w.dragWindow = nil }()

	if fw := w.Desktop().FocusedWindow(); fw != nil && button == tcell.Button1 && mods == 0 {
		ds := fw.dragState
		fw.dragState = 0
		screenPos0 := fw.dragScreenPos0
		winPos0 := fw.dragWinPos0
		winSize0 := fw.dragWinSize0
		dx := screenPos.X - screenPos0.X
		dy := screenPos.Y - screenPos0.Y

		switch ds {
		case dragPos:
			fw.SetPosition(Position{winPos0.X + dx, winPos0.Y + dy})
			return
		case dragRightSize:
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width+dx), winSize0.Height})
			return
		case dragLeftSize:
			if dx > winSize0.Width {
				dx = winSize0.Width - 1
			}
			fw.SetPosition(Position{winPos0.X + dx, winPos0.Y})
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width-dx), winSize0.Height})
			return
		case dragBottomSize:
			fw.SetSize(Size{winSize0.Width, mathutil.Max(1, winSize0.Height+dy)})
			return
		case dragLRC:
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width+dx), mathutil.Max(1, winSize0.Height+dy)})
			return
		case dragURC:
			if dy > winSize0.Height {
				dy = winSize0.Height - 1
			}
			fw.SetPosition(Position{winPos0.X, winPos0.Y + dy})
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width+dx), mathutil.Max(1, winSize0.Height-dy)})
			return
		case dragLLC:
			if dx > winSize0.Width {
				dx = winSize0.Width - 1
			}
			fw.SetPosition(Position{winPos0.X + dx, winPos0.Y})
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width-dx), mathutil.Max(1, winSize0.Height+dy)})
			return
		case dragULC:
			if dx > winSize0.Width {
				dx = winSize0.Width - 1
			}
			if dy > winSize0.Height {
				dy = winSize0.Height - 1
			}
			fw.SetPosition(Position{winPos0.X + dx, winPos0.Y + dy})
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width-dx), mathutil.Max(1, winSize0.Height-dy)})
			return
		default:
			if fw == w.dragWindow {
				fw.onDrop.Handle(fw, button, screenPos, w.dragWindowPos, mods)
				return
			}
		}
	}

	w.event(
		screenPos,
		func(w *Window, winPos Position) {
			w.onDrop.Handle(w, button, screenPos, winPos, mods)
		},
		func(w *Window, winPos Position) {
			w.onDrop.Handle(w, button, screenPos, winPos, mods)
		},
		true,
	)
}
func (w *Window) mouseMove(button tcell.ButtonMask, screenPos Position, mods tcell.ModMask) {
	if fw := w.Desktop().FocusedWindow(); fw != nil {
		ds := fw.dragState
		screenPos0 := fw.dragScreenPos0
		winPos0 := fw.dragWinPos0
		winSize0 := fw.dragWinSize0
		dx := screenPos.X - screenPos0.X
		dy := screenPos.Y - screenPos0.Y

		switch ds {
		case dragPos:
			fw.SetPosition(Position{winPos0.X + dx, winPos0.Y + dy})
			return
		case dragRightSize:
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width+dx), winSize0.Height})
			return
		case dragLeftSize:
			if dx > winSize0.Width {
				dx = winSize0.Width - 1
			}
			fw.SetPosition(Position{winPos0.X + dx, winPos0.Y})
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width-dx), winSize0.Height})
			return
		case dragBottomSize:
			fw.SetSize(Size{winSize0.Width, mathutil.Max(1, winSize0.Height+dy)})
			return
		case dragLRC:
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width+dx), mathutil.Max(1, winSize0.Height+dy)})
			return
		case dragURC:
			if dy > winSize0.Height {
				dy = winSize0.Height - 1
			}
			fw.SetPosition(Position{winPos0.X, winPos0.Y + dy})
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width+dx), mathutil.Max(1, winSize0.Height-dy)})
			return
		case dragLLC:
			if dx > winSize0.Width {
				dx = winSize0.Width - 1
			}
			fw.SetPosition(Position{winPos0.X + dx, winPos0.Y})
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width-dx), mathutil.Max(1, winSize0.Height+dy)})
			return
		case dragULC:
			if dx > winSize0.Width {
				dx = winSize0.Width - 1
			}
			if dy > winSize0.Height {
				dy = winSize0.Height - 1
			}
			fw.SetPosition(Position{winPos0.X + dx, winPos0.Y + dy})
			fw.SetSize(Size{mathutil.Max(1, winSize0.Width-dx), mathutil.Max(1, winSize0.Height-dy)})
			return
		default:
			if fw == w.dragWindow {
				fw.onMouseMove.Handle(fw, button, screenPos, w.dragWindowPos, mods)
				return
			}
		}
	}

	w.event(
		screenPos,
		func(w *Window, winPos Position) {
			w.onMouseMove.Handle(w, button, screenPos, winPos, mods)
		},
		func(w *Window, winPos Position) {},
		false,
	)
}

// paint asks w to render an area.
func (w *Window) paint(area Rectangle) {
	d := w.Desktop()
	if area.IsZero() || !area.Clip(Rectangle{Size: w.size}) || d != App.Desktop() {
		return
	}

	if d.updateLevel != 0 {
		for {
			p := w.Parent()
			if p == nil {
				d.invalidated.join(area)
				return
			}

			area.Position = area.add(w.Position()).add(p.ClientPosition().sub(p.Origin()))
			if !area.Clip(p.ClientArea()) {
				return
			}

			w = p
		}
	}

	a0 := w.Area()
	if a := a0; a.Clip(area) {
		w.onClearBorders.Handle(w, PaintContext{a, a0.Position, Position{}})
	}

	a0 = w.BorderTopArea()
	if a := a0; a.Clip(area) {
		w.onPaintBorderTop.Handle(w, PaintContext{a, a0.Position, Position{}})
	}

	if !a0.IsZero() && w.Title() != "" {
		a0.X++
		a0.Width--
		if w.CloseButton() {
			a0.Width -= closeButtonOffset
		}
		a0.Height = 1
		if a := a0; a.Clip(area) {
			w.onPaintTitle.Handle(w, PaintContext{a, a0.Position, Position{}})
		}
	}

	a0 = w.BorderLeftArea()
	if a := a0; a.Clip(area) {
		w.onPaintBorderLeft.Handle(w, PaintContext{a, a0.Position, Position{}})
	}

	a0 = w.ClientArea()
	if a := a0; a.Clip(area) {
		ctx := PaintContext{a, a0.Position, Position{}}
		w.onClearClientArea.Handle(w, ctx)
	}

	a0 = w.ClientArea()
	if a := a0; a.Clip(area) {
		a.Position = a.add(w.view)
		ctx := PaintContext{a, a0.Position, w.view}
		w.onPaintClientArea.Handle(w, ctx)
		w.onPaintChildren.Handle(w, ctx)
	}

	a0 = w.BorderRightArea()
	if a := a0; a.Clip(area) {
		w.onPaintBorderRight.Handle(w, PaintContext{a, a0.Position, Position{}})
	}

	a0 = w.BorderBottomArea()
	if a := a0; a.Clip(area) {
		w.onPaintBorderBottom.Handle(w, PaintContext{a, a0.Position, Position{}})
	}
}

func (w *Window) print(x, y int, style tcell.Style, s string) {
	if s == "" {
		return
	}

	if w.ctx.IsZero() { // Zero sized window or not in OnPaint.
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

func (w *Window) bringChildWindowToFront(c *Window) {
	if w == nil {
		return
	}

	if p := w.Parent(); p != nil {
		p.bringChildWindowToFront(w)
	}
	for i, v := range w.children {
		if v == c {
			if i == len(w.children)-1 { // Already in front.
				return
			}

			copy(w.children[i:], w.children[i+1:])
			w.children[len(w.children)-1] = c
			break
		}
	}
	w.InvalidateClientArea(Rectangle{c.Position(), c.Size()})
}

func (w *Window) removeChild(ch *Window) {
	for i, v := range w.children {
		if v == ch {
			copy(w.children[i:], w.children[i+1:])
			w.children = w.children[:len(w.children)-1]
			break
		}
	}
}

func (w *Window) closeButtonArea() (r Rectangle) {
	if w.BorderTop() > 0 {
		r.X = w.size.Width - closeButtonOffset
		r.Width = closeButtonWidth
		r.Height = 1
	}
	return r
}

func (w *Window) topBorderDragMoveArea() (r Rectangle) {
	r = w.BorderTopArea()
	if !r.IsZero() {
		r.X++
		r.Width -= 2
		r.Height = 1
	}
	return r
}

func (w *Window) leftBorderDragResizeArea() (r Rectangle) {
	r = w.BorderLeftArea()
	if !r.IsZero() {
		r.Y++
		r.Height -= 2
		r.Width = 1
	}
	return r
}

func (w *Window) rightBorderDragResizeArea() (r Rectangle) {
	r = w.BorderRightArea()
	if !r.IsZero() {
		r.Y++
		r.Height -= 2
		r.X += r.Width - 1
	}
	return r
}

func (w *Window) bottomBorderDragResizeArea() (r Rectangle) {
	r = w.BorderBottomArea()
	if !r.IsZero() {
		r.X++
		r.Width -= 2
		r.Y += r.Height - 1
		r.Height = 1
	}
	return r
}

func (w *Window) borderLRCArea() (r Rectangle) {
	r = w.BorderRightArea()
	if !r.IsZero() {
		r.X += r.Width - 1
		r.Y += r.Height - 1
		r.Width = 1
		r.Height = 1
	}
	return r
}

func (w *Window) borderURCArea() (r Rectangle) {
	r = w.BorderRightArea()
	if !r.IsZero() {
		r.X += r.Width - 1
		r.Width = 1
		r.Height = 1
	}
	return r
}

func (w *Window) borderLLCArea() (r Rectangle) {
	r = w.BorderBottomArea()
	if !r.IsZero() {
		r.Y += r.Height - 1
		r.Width = 1
		r.Height = 1
	}
	return r
}

func (w *Window) borderULCArea() (r Rectangle) {
	r = w.BorderTopArea()
	if !r.IsZero() {
		r.Width = 1
		r.Height = 1
	}
	return r
}

// ----------------------------------------------------------------------------

// Area returns the area of the window.
func (w *Window) Area() Rectangle { return Rectangle{Size: w.size} }

// BorderBottom returns the height of the bottom border.
func (w *Window) BorderBottom() int { return w.borderBottom }

// BorderBottomArea returns the area of the bottom border.
func (w *Window) BorderBottomArea() (r Rectangle) {
	r.Y = w.size.Height - w.borderBottom
	r.Width = w.size.Width
	r.Height = w.borderBottom
	return r
}

// BorderLeft returns the width of the left border.
func (w *Window) BorderLeft() int { return w.borderLeft }

// BorderLeftArea returns the area of the left border.
func (w *Window) BorderLeftArea() (r Rectangle) {
	r.Width = w.borderLeft
	r.Height = w.size.Height
	return r
}

// BorderRight returns the width of the right border.
func (w *Window) BorderRight() int { return w.borderRight }

// BorderRightArea returns the area of the right border.
func (w *Window) BorderRightArea() (r Rectangle) {
	r.X = w.size.Width - w.borderRight
	r.Width = w.borderRight
	r.Height = w.size.Height
	return r
}

// BorderStyle returns the border style.
func (w *Window) BorderStyle() Style { return w.style.Border }

// BorderTop returns the height of the top border.
func (w *Window) BorderTop() int { return w.borderTop }

// BorderTopArea returns the area of the top border.
func (w *Window) BorderTopArea() (r Rectangle) {
	r.Width = w.size.Width
	r.Height = w.borderTop
	return r
}

// BringToFront puts a child window on top of all its siblings. The method has
// no effect if w is a root window.
func (w *Window) BringToFront() { w.Parent().bringChildWindowToFront(w) }

// Child returns the nth child window or nil if no such exists.
func (w *Window) Child(n int) (r *Window) {
	if n < len(w.children) {
		return w.children[n]
	}

	return nil
}

// Children returns the number of child windows.
func (w *Window) Children() (r int) {
	r = len(w.children)
	return r
}

// ClientArea returns the client area.
func (w *Window) ClientArea() Rectangle { return w.clientArea }

// ClientPosition returns the position of the client area relative to w.
func (w *Window) ClientPosition() Position { return w.clientArea.Position }

// ClientSize returns the size of the client area.
func (w *Window) ClientSize() Size { return w.clientArea.Size }

// ClientAreaStyle returns the client area style.
func (w *Window) ClientAreaStyle() Style { return w.style.ClientArea }

// Close closes w.
func (w *Window) Close() {
	w.onClose.handle(w)
	w.SetFocus(false)
	for w.Children() != 0 {
		if c := w.Child(0); c != nil {
			c.Close()
		}
	}
	if p := w.Parent(); p != nil {
		p.removeChild(w)
		p.InvalidateClientArea(p.ClientArea())
	}

	w.onClearBorders.Clear()
	w.onClearClientArea.Clear()
	w.onClick.Clear()
	w.onClickBorder.Clear()
	w.onClose.clear()
	w.onDoubleClick.Clear()
	w.onDoubleClickBorder.Clear()
	w.onDrag.Clear()
	w.onDragBorder.Clear()
	w.onDrop.Clear()
	w.onKey.clear()
	w.onMouseMove.Clear()
	w.onPaintBorderBottom.Clear()
	w.onPaintBorderLeft.Clear()
	w.onPaintBorderRight.Clear()
	w.onPaintBorderTop.Clear()
	w.onPaintChildren.Clear()
	w.onPaintClientArea.Clear()
	w.onPaintTitle.Clear()
	w.onSetBorderBotom.Clear()
	w.onSetBorderLeft.Clear()
	w.onSetBorderRight.Clear()
	w.onSetBorderStyle.Clear()
	w.onSetBorderTop.Clear()
	w.onSetClientAreaStyle.Clear()
	w.onSetClientSize.Clear()
	w.onSetCloseButton.Clear()
	w.onSetFocus.Clear()
	w.onSetFocusedWindow.clear()
	w.onSetOrigin.Clear()
	w.onSetPosition.Clear()
	w.onSetSelection.clear()
	w.onSetSize.Clear()
	w.onSetStyle.clear()
	w.onSetTitle.clear()
}

// CloseButton returns whether the window shows a close button.
func (w *Window) CloseButton() bool { return w.closeButton }

// Desktop returns which Desktop w appears on.
func (w *Window) Desktop() *Desktop { return w.desktop }

// Focus returns wheter the window is focused.
func (w *Window) Focus() bool { return w.focus }

// Invalidate marks a window area for repaint.
func (w *Window) Invalidate(area Rectangle) {
	if !area.Clip(Rectangle{Size: w.size}) {
		return
	}

	w.BeginUpdate()
	w.paint(area)
	w.EndUpdate()
}

// InvalidateClientArea marks an area of the client area for repaint.
func (w *Window) InvalidateClientArea(area Rectangle) {
	area.Position = area.Position.add(w.ClientPosition()).sub(w.Origin())
	if !area.Clip(w.clientArea) {
		return
	}

	w.BeginUpdate()
	w.paint(area)
	w.EndUpdate()
}

// NewChild creates a child window.
func (w *Window) NewChild(area Rectangle) *Window {
	w.BeginUpdate()
	c := newWindow(w.desktop, w, App.ChildWindowStyle())
	w.children = append(w.children, c)
	c.SetBorderTop(1)
	c.SetBorderLeft(1)
	c.SetBorderRight(1)
	c.SetBorderBottom(1)
	c.SetPosition(area.Position)
	c.SetSize(area.Size)
	w.EndUpdate()
	return c
}

// OnClick sets a mouse click event handler. When the event handler is removed,
// finalize is called, if not nil.
func (w *Window) OnClick(h OnMouseHandler, finalize func()) {
	AddOnMouseHandler(&w.onClick, h, finalize)
}

// OnClickBorder sets a mouse click border event handler. When the event
// handler is removed, finalize is called, if not nil.
func (w *Window) OnClickBorder(h OnMouseHandler, finalize func()) {
	AddOnMouseHandler(&w.onClickBorder, h, finalize)
}

// OnClose sets a window close event handler. When the event handler is
// removed, finalize is called, if not nil.
func (w *Window) OnClose(h OnCloseHandler, finalize func()) {
	addOnCloseHandler(&w.onClose, h, finalize)
}

// OnDoubleClick sets a mouse double click event handler. When the event
// handler is removed, finalize is called, if not nil.
func (w *Window) OnDoubleClick(h OnMouseHandler, finalize func()) {
	AddOnMouseHandler(&w.onDoubleClick, h, finalize)
}

// OnDoubleClickBorder sets a mouse double click border event handler. When the
// event handler is removed, finalize is called, if not nil.
func (w *Window) OnDoubleClickBorder(h OnMouseHandler, finalize func()) {
	AddOnMouseHandler(&w.onDoubleClickBorder, h, finalize)
}

// OnDrag sets a mouse drag event handler. When the event handler is removed,
// finalize is called, if not nil.
func (w *Window) OnDrag(h OnMouseHandler, finalize func()) {
	AddOnMouseHandler(&w.onDrag, h, finalize)
}

// OnDragBorder sets a mouse drag border event handler. When the event handler
// is removed, finalize is called, if not nil.
func (w *Window) OnDragBorder(h OnMouseHandler, finalize func()) {
	AddOnMouseHandler(&w.onDragBorder, h, finalize)
}

// OnDrop sets a mouse drop event handler. When the event handler is removed,
// finalize is called, if not nil.
func (w *Window) OnDrop(h OnMouseHandler, finalize func()) {
	AddOnMouseHandler(&w.onDrop, h, finalize)
}

// OnKey sets a key event handler. When the event handler is removed, finalize
// is called, if not nil.
func (w *Window) OnKey(h OnKeyHandler, finalize func()) {
	addOnKeyHandler(&w.onKey, h, finalize)
}

// OnMouseMove sets a mouse move event handler. When the event handler is
// removed, finalize is called, if not nil.
func (w *Window) OnMouseMove(h OnMouseHandler, finalize func()) {
	AddOnMouseHandler(&w.onMouseMove, h, finalize)
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
	AddOnPaintHandler(&w.onPaintClientArea, h, finalize)
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
	AddOnPaintHandler(&w.onPaintBorderBottom, h, finalize)
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
	AddOnPaintHandler(&w.onPaintBorderLeft, h, finalize)
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
	AddOnPaintHandler(&w.onPaintBorderRight, h, finalize)
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
	AddOnPaintHandler(&w.onPaintBorderTop, h, finalize)
}

// OnPaintTitle sets a window title paint handler. When the event handler is
// removed, finalize is called, if not nil. Example:
//
//	if s := w.Title(); s != "" {
//		w.Printf(0, 0, w.Style().Title, " %s ", s)
//	}
func (w *Window) OnPaintTitle(h OnPaintHandler, finalize func()) {
	AddOnPaintHandler(&w.onPaintTitle, h, finalize)
}

// OnSetBorderBottom sets a handler invoked on SetBorderBottom. When the event
// handler is removed, finalize is called, if not nil.
func (w *Window) OnSetBorderBottom(h OnSetIntHandler, finalize func()) {
	AddOnSetIntHandler(&w.onSetBorderBotom, h, finalize)
}

// OnSetBorderLeft sets a handler invoked on SetBorderLeft. When the event
// handler is removed, finalize is called, if not nil.
func (w *Window) OnSetBorderLeft(h OnSetIntHandler, finalize func()) {
	AddOnSetIntHandler(&w.onSetBorderLeft, h, finalize)
}

// OnSetBorderRight sets a handler invoked on SetBorderRight. When the event
// handler is removed, finalize is called, if not nil.
func (w *Window) OnSetBorderRight(h OnSetIntHandler, finalize func()) {
	AddOnSetIntHandler(&w.onSetBorderRight, h, finalize)
}

// OnSetBorderStyle sets a handler invoked on SetBorderStyle. When the event
// handler is removed, finalize is called, if not nil.
func (w *Window) OnSetBorderStyle(h OnSetStyleHandler, finalize func()) {
	AddOnSetStyleHandler(&w.onSetBorderStyle, h, finalize)
}

// OnSetBorderTop sets a handler invoked on SetBorderTop. When the event
// handler is removed, finalize is called, if not nil.
func (w *Window) OnSetBorderTop(h OnSetIntHandler, finalize func()) {
	AddOnSetIntHandler(&w.onSetBorderTop, h, finalize)
}

// OnSetClientAreaStyle sets a handler invoked on SetClientAreaStyle. When the
// event handler is removed, finalize is called, if not nil.
func (w *Window) OnSetClientAreaStyle(h OnSetStyleHandler, finalize func()) {
	AddOnSetStyleHandler(&w.onSetClientAreaStyle, h, finalize)
}

// OnSetClientSize sets a handler invoked on SetClientSize. When the event
// handler is removed, finalize is called, if not nil.
func (w *Window) OnSetClientSize(h OnSetSizeHandler, finalize func()) {
	AddOnSetSizeHandler(&w.onSetClientSize, h, finalize)
}

// OnSetCloseButton sets a handler invoked on SetCloseButton. When the event
// handler is removed, finalize is called, if not nil.
func (w *Window) OnSetCloseButton(h OnSetBoolHandler, finalize func()) {
	AddOnSetBoolHandler(&w.onSetCloseButton, h, finalize)
}

// OnSetFocus sets a handler invoked on SetFocus. When the event handler is
// removed, finalize is called, if not nil.
func (w *Window) OnSetFocus(h OnSetBoolHandler, finalize func()) {
	AddOnSetBoolHandler(&w.onSetFocus, h, finalize)
}

// OnSetOrigin sets a handler invoked on SetOrigin. When the event handler
// is removed, finalize is called, if not nil.
func (w *Window) OnSetOrigin(h OnSetPositionHandler, finalize func()) {
	AddOnSetPositionHandler(&w.onSetOrigin, h, finalize)
}

// OnSetPosition sets a handler invoked on SetPosition. When the event handler
// is removed, finalize is called, if not nil.
func (w *Window) OnSetPosition(h OnSetPositionHandler, finalize func()) {
	AddOnSetPositionHandler(&w.onSetPosition, h, finalize)
}

// OnSetSize sets a handler invoked on SetSize. When the event handler is
// removed, finalize is called, if not nil.
func (w *Window) OnSetSize(h OnSetSizeHandler, finalize func()) {
	AddOnSetSizeHandler(&w.onSetSize, h, finalize)
}

// OnSetStyle sets a handler invoked on SetStyle. When the event handler is
// removed, finalize is called, if not nil.
func (w *Window) OnSetStyle(h OnSetWindowStyleHandler, finalize func()) {
	addOnSetWindowStyleHandler(&w.onSetStyle, h, finalize)
}

// OnSetTitle sets a handler invoked on SetTitle. When the event handler is
// removed, finalize is called, if not nil.
func (w *Window) OnSetTitle(h OnSetStringHandler, finalize func()) {
	addOnSetStringHandler(&w.onSetTitle, h, finalize)
}

// Origin returns the window's origin..
func (w *Window) Origin() Position { return w.view }

// Printf prints format with arguments at x, y. Calling this method outside of
// an OnPaint handler is ignored.
//
// Special handling:
//
//	'\t'	x, y = x + 8 - x%8, y
//	'\n'	x, y = 0, y+1
//	'\r'	x, y = 0, y
func (w *Window) Printf(x, y int, style Style, format string, arg ...interface{}) {
	if w.ctx.IsZero() { // Zero sized window or not in OnPaint.
		return
	}

	w.print(x, y, style.TCellStyle(), fmt.Sprintf(format, arg...))
}

// Parent returns the window's parent. Root windows have nil parent.
func (w *Window) Parent() *Window { return w.parent }

// Position returns the window position relative to its parent.
func (w *Window) Position() Position { return w.position }

// RemoveOnClick undoes the most recent OnClick call. The function will panic if
// there is no handler set.
func (w *Window) RemoveOnClick() { RemoveOnMouseHandler(&w.onClick) }

// RemoveOnClickBorder undoes the most recent OnClickBorder call. The function
// will panic if there is no handler set.
func (w *Window) RemoveOnClickBorder() { RemoveOnMouseHandler(&w.onClickBorder) }

// RemoveOnClose undoes the most recent OnClose call. The function will panic
// if there is no handler set.
func (w *Window) RemoveOnClose() { removeOnCloseHandler(&w.onClose) }

// RemoveOnDoubleClick undoes the most recent OnDoubleClick call. The function
// will panic if there is no handler set.
func (w *Window) RemoveOnDoubleClick() { RemoveOnMouseHandler(&w.onDoubleClick) }

// RemoveOnDoubleClickBorder undoes the most recent OnDoubleClickBorder call.
// The function will panic if there is no handler set.
func (w *Window) RemoveOnDoubleClickBorder() { RemoveOnMouseHandler(&w.onDoubleClickBorder) }

// RemoveOnDrag undoes the most recent OnDrag call. The function will panic if
// there is no handler set.
func (w *Window) RemoveOnDrag() { RemoveOnMouseHandler(&w.onDrag) }

// RemoveOnDragBorder undoes the most recent OnDragBorder call. The function
// will panic if there is no handler set.
func (w *Window) RemoveOnDragBorder() { RemoveOnMouseHandler(&w.onDragBorder) }

// RemoveOnDrop undoes the most recent OnDrop call. The function will panic if
// there is no handler set.
func (w *Window) RemoveOnDrop() { RemoveOnMouseHandler(&w.onDrop) }

// RemoveOnKey undoes the most recent OnKey call. The function will panic if
// there is no handler set.
func (w *Window) RemoveOnKey() { removeOnKeyHandler(&w.onKey) }

// RemoveOnMouseMove undoes the most recent OnMouseMove call. The function will
// panic if there is no handler set.
func (w *Window) RemoveOnMouseMove() { RemoveOnMouseHandler(&w.onMouseMove) }

// RemoveOnPaintClientArea undoes the most recent OnPaintClientArea call. The
// function will panic if there is no handler set.
func (w *Window) RemoveOnPaintClientArea() { RemoveOnPaintHandler(&w.onPaintClientArea) }

// RemoveOnPaintBorderBottom undoes the most recent OnPaintBorderBottom call.
// The function will panic if there is no handler set.
func (w *Window) RemoveOnPaintBorderBottom() { RemoveOnPaintHandler(&w.onPaintBorderBottom) }

// RemoveOnPaintBorderLeft undoes the most recent OnPaintBorderLeft call.
// The function will panic if there is no handler set.
func (w *Window) RemoveOnPaintBorderLeft() { RemoveOnPaintHandler(&w.onPaintBorderLeft) }

// RemoveOnPaintBorderRight undoes the most recent OnPaintBorderRight call.
// The function will panic if there is no handler set.
func (w *Window) RemoveOnPaintBorderRight() { RemoveOnPaintHandler(&w.onPaintBorderRight) }

// RemoveOnPaintBorderTop undoes the most recent OnPaintBorderTop call.
// The function will panic if there is no handler set.
func (w *Window) RemoveOnPaintBorderTop() { RemoveOnPaintHandler(&w.onPaintBorderTop) }

// RemoveOnPaintTitle undoes the most recent OnPaintTitle call.  The function
// will panic if there is no handler set.
func (w *Window) RemoveOnPaintTitle() { RemoveOnPaintHandler(&w.onPaintTitle) }

// RemoveOnSetBorderBottom undoes the most recent OnSetBorderBottom call. The
// function will panic if there is no handler set.
func (w *Window) RemoveOnSetBorderBottom() { RemoveOnSetIntHandler(&w.onSetBorderBotom) }

// RemoveOnSetBorderLeft undoes the most recent OnSetBorderLeft call. The
// function will panic if there is no handler set.
func (w *Window) RemoveOnSetBorderLeft() { RemoveOnSetIntHandler(&w.onSetBorderLeft) }

// RemoveOnSetBorderRight undoes the most recent OnSetBorderRight call. The
// function will panic if there is no handler set.
func (w *Window) RemoveOnSetBorderRight() { RemoveOnSetIntHandler(&w.onSetBorderRight) }

// RemoveOnSetBorderStyle undoes the most recent OnSetBorderStyle call. The
// function will panic if there is no handler set.
func (w *Window) RemoveOnSetBorderStyle() { RemoveOnSetStyleHandler(&w.onSetBorderStyle) }

// RemoveOnSetBorderTop undoes the most recent OnSetBorderTop call. The
// function will panic if there is no handler set.
func (w *Window) RemoveOnSetBorderTop() { RemoveOnSetIntHandler(&w.onSetBorderTop) }

// RemoveOnSetClientAreaStyle undoes the most recent OnSetClientAreaStyle call.
// The function will panic if there is no handler set.
func (w *Window) RemoveOnSetClientAreaStyle() { RemoveOnSetStyleHandler(&w.onSetClientAreaStyle) }

// RemoveOnSetClientSize undoes the most recent OnSetClientSize call. The
// function will panic if there is no handler set.
func (w *Window) RemoveOnSetClientSize() { RemoveOnSetSizeHandler(&w.onSetClientSize) }

// RemoveOnSetCloseButton undoes the most recent OnSetCloseButton call. The
// function will panic if there is no handler set.
func (w *Window) RemoveOnSetCloseButton() { RemoveOnSetBoolHandler(&w.onSetCloseButton) }

// RemoveOnSetFocus undoes the most recent OnSetFocus call. The function will
// panic if there is no handler set.
func (w *Window) RemoveOnSetFocus() { RemoveOnSetBoolHandler(&w.onSetFocus) }

// RemoveOnSetOrigin undoes the most recent OnSetOrigin call. The function
// will panic if there is no handler set.
func (w *Window) RemoveOnSetOrigin() { RemoveOnSetPositionHandler(&w.onSetOrigin) }

// RemoveOnSetPosition undoes the most recent OnSetPosition call. The function
// will panic if there is no handler set.
func (w *Window) RemoveOnSetPosition() { RemoveOnSetPositionHandler(&w.onSetPosition) }

// RemoveOnSetSize undoes the most recent OnSetSize call. The function will
// panic if there is no handler set.
func (w *Window) RemoveOnSetSize() { RemoveOnSetSizeHandler(&w.onSetSize) }

// RemoveOnSetStyle undoes the most recent OnSetStyle call. The function will
// panic if there is no handler set.
func (w *Window) RemoveOnSetStyle() { removeOnSetWindowStyleHandler(&w.onSetStyle) }

// RemoveOnSetTitle undoes the most recent OnSetTitle call. The function will
// panic if there is no handler set.
func (w *Window) RemoveOnSetTitle() { removeOnSetStringHandler(&w.onSetTitle) }

// Rendered returns how long the last desktop rendering took. Valid only for
// desktop's root window.
func (w *Window) Rendered() time.Duration { return w.rendered }

// SetBorderBottom sets the height of the bottom border.
func (w *Window) SetBorderBottom(v int) { w.onSetBorderBotom.Handle(w, &w.borderBottom, v) }

// SetBorderLeft sets the width of the left border.
func (w *Window) SetBorderLeft(v int) { w.onSetBorderLeft.Handle(w, &w.borderLeft, v) }

// SetBorderRight sets the width of the right border.
func (w *Window) SetBorderRight(v int) { w.onSetBorderRight.Handle(w, &w.borderRight, v) }

// SetBorderStyle sets the border style.
func (w *Window) SetBorderStyle(s Style) { w.onSetBorderStyle.Handle(w, &w.style.Border, s) }

// SetBorderTop sets the height of the top border.
func (w *Window) SetBorderTop(v int) { w.onSetBorderTop.Handle(w, &w.borderTop, v) }

// SetCell renders a single character cell. Calling this method outside of an
// OnPaint* handler is ignored.
func (w *Window) SetCell(x, y int, mainc rune, combc []rune, style tcell.Style) {
	w.BeginUpdate()
	w.setCell(Position{x, y}, mainc, combc, style)
	w.EndUpdate()
}

// SetClientAreaStyle sets the client area style.
func (w *Window) SetClientAreaStyle(s Style) { w.onSetClientAreaStyle.Handle(w, &w.style.ClientArea, s) }

// SetClientSize sets the size of the client area.
func (w *Window) SetClientSize(s Size) { w.onSetClientSize.Handle(w, &w.clientArea.Size, s) }

// SetCloseButton sets whether the window shows a close button.
func (w *Window) SetCloseButton(v bool) {
	if w.parent != nil {
		w.onSetCloseButton.Handle(w, &w.closeButton, v)
	}
}

// SetFocus sets whether the window is focused.
func (w *Window) SetFocus(v bool) { w.onSetFocus.Handle(w, &w.focus, v) }

// SetOrigin sets the origin of the window. By default the origin of a window
// is (0, 0).  When a paint handler is invoked the window's origin is
// subtracted from the coordinates the handler paints to. Also, the
// PaintContext.Rectangle value passed to the handler is adjusted accordingly.
//
// Setting origin other than (0, 0) provides a view into the content created by
// the respective paint handlers. The mechanism aids in displaying scrolling
// content.
//
// Negative values of p.X or p.Y are ignored.
func (w *Window) SetOrigin(p Position) {
	w.onSetOrigin.Handle(w, &w.view, Position{X: mathutil.Max(p.X, 0), Y: mathutil.Max(p.Y, 0)})
}

// SetPosition sets the window position relative to its parent.
func (w *Window) SetPosition(p Position) {
	if w.parent != nil {
		w.onSetPosition.Handle(w, &w.position, p)
	}
}

// SetSize sets the window size.
func (w *Window) SetSize(s Size) {
	if w.parent != nil {
		w.setSize(s)
	}
}

// SetStyle sets the window style.
func (w *Window) SetStyle(s WindowStyle) { w.onSetStyle.handle(w, &w.style, s) }

// SetTitle sets the window title.
func (w *Window) SetTitle(s string) { w.onSetTitle.handle(w, &w.title, s) }

// Size returns the window size.
func (w *Window) Size() Size { return w.size }

// Style returns the window style.
func (w *Window) Style() WindowStyle { return w.style }

// Title returns the window title.
func (w *Window) Title() string { return w.title }
