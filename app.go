// Copyright 2015 The WM Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wm

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/encoding"
)

const (
	anyButton = tcell.Button8<<1 - 1
)

// Application represents an interactive terminal application.
//
// All exported methods of an Application are safe for concurrent access by
// multiple goroutines. However, concurrent calls of Application methods that
// mutate its state may produce flicker or corrupted screen rendering.
type Application struct {
	click             time.Duration             //
	desktop           *Desktop                  //
	doubleClick       time.Duration             //
	mouseButtonFSMs   [8]*mouseButtonFSM        //
	mouseButtonsState tcell.ButtonMask          //
	mouseX            int                       //
	mouseY            int                       //
	mu                sync.Mutex                //
	onKey             *onKeyHandlerList         //
	onSetClick        *onSetDurationHandlerList //
	onSetDesktop      *onSetDesktopHandlerList  //
	onSetDoubleClick  *onSetDurationHandlerList //
	onSetSize         *onSetSizeHandlerList     //
	onceFinalize      sync.Once                 //
	onceWait          sync.Once                 //
	screen            tcell.Screen              //
	size              Size                      //
	terminated        bool                      // quitMessage processed
	theme             *Theme                    //
	updateLevel       int32                     //
	wait              chan error                //
}

// NewApplication returns a newly created Application or an error, if any.
//
//	// Skeleton example.
//	func main() {
//		app, err := wm.NewApplication(theme)
//		if err != nil {
//			log.Fatalf("error: %v", err)
//		}
//
//		defer app.Finalize()
//
//		...
//
//		if err := app.Wait(); err != nil {
//			log.Println("error: %v", err)
//		}
//	}
//
// Calling this function more than once will panic.
func NewApplication(theme *Theme) (*Application, error) {
	done := false
	var app *Application
	var err error
	onceNewApplication.Do(func() {
		app, err = newApplication(nil, theme)
		done = true
	})
	if !done {
		panic("NewApplication called more than once")
	}

	return app, err
}

func newApplication(screen tcell.Screen, t *Theme) (*Application, error) {
	encoding.Register()
	var err error
	if screen == nil {
		if screen, err = tcell.NewScreen(); err != nil {
			return nil, err
		}
	}

	if err = screen.Init(); err != nil {
		return nil, err
	}

	var size Size
	size.Width, size.Height = screen.Size()
	theme := *t
	app = &Application{
		click:       150 * time.Millisecond,
		doubleClick: 120 * time.Millisecond,
		screen:      screen,
		size:        size,
		theme:       &theme,
		wait:        make(chan error, 1),
	}

	mask := tcell.Button1
	for i := range app.mouseButtonFSMs {
		app.mouseButtonFSMs[i] = newMouseButton(mask)
		mask <<= 1
	}
	app.screen.EnableMouse()
	go app.handleEvents()
	app.OnSetDesktop(app.onSetDesktopHandler, nil)
	app.OnSetSize(app.onSetSizeHandler, nil)
	return app, nil
}

func (a *Application) handleEvents() {
	for {
		ev := a.screen.PollEvent()
		if ev == nil {
			return
		}

		switch e := ev.(type) {
		case *tcell.EventResize:
			a.setSize(newSize(e.Size()))
		case *tcell.EventKey:
			a.onKey.handle(nil, e.Key(), e.Modifiers(), e.Rune())
		case *tcell.EventMouse:
			x, y := e.Position()
			if x != a.mouseX || y != a.mouseY {
				a.mouseX = x
				a.mouseY = y
				app.screen.PostEvent(newEventMouse(mouseMove, 0, e.Modifiers(), Position{x, y}))
			}
			if b := e.Buttons() & anyButton; b != a.mouseButtonsState {
				diff := b ^ a.mouseButtonsState
				a.mouseButtonsState = b
				x := 0
				for diff != 0 {
					a.mouseButtonFSMs[x].post(e)
					diff >>= 1
					x++
				}
			}
		case *eventMouse:
			w := a.Desktop().Root()
			switch e.kind {
			case mouseDrag:
				w.drag(e.button, e.Position, e.mods)
			case mouseDrop:
				w.drop(e.button, e.Position, e.mods)
			case mouseClick:
				w.click(e.button, e.Position, e.mods)
			case mouseDoubleClick:
				w.doubleClick(e.button, e.Position, e.mods)
			case mouseMove:
				w.mouseMove(e.Position, e.mods)
			default:
				panic(fmt.Errorf("%v", e.kind))
			}
			e.dispose()
		case *eventFunc:
			e.f()
			e.dispose()
		default:
			panic(fmt.Errorf("%T", e))
		}
	}
}

func (a *Application) onSetDesktopHandler(_ *Window, prev OnSetDesktopHandler, dst **Desktop, src *Desktop) {
	if prev != nil {
		prev(nil, nil, dst, src)
	} else {
		*dst = src
	}

	w := a.Desktop().Root()
	w.setSize(a.Size())
	w.Invalidate(w.Area())
}

func (a *Application) onSetSizeHandler(_ *Window, prev OnSetSizeHandler, dst *Size, src Size) {
	if prev != nil {
		prev(nil, nil, dst, src)
	} else {
		*dst = src
	}

	d := a.Desktop()
	if d == nil {
		return
	}

	sz := a.Size()
	w := d.Root()
	w.setSize(sz)
}

func (a *Application) paintSelection() {
	d := a.Desktop()
	if d == nil {
		return
	}

	area := d.Selection()
	if area.IsZero() {
		return
	}

	o := area.Position
	for y := 0; y < area.Height; y++ {
		sy := o.Y + y
		fx := true
		for x := 0; x < area.Width; x++ {
			sx := o.X + x
			if fx {
				_, _, _, width := a.screen.GetContent(sx-1, sy)
				if width == 2 {
					sx--
					x--
				}
			}
			fx = false
			mainc, combc, style, width := a.screen.GetContent(sx, sy)
			style ^= tcell.Style(tcell.AttrReverse)
			a.screen.SetContent(sx, sy, mainc, combc, style)
			if width == 2 {
				x++
			}
		}
	}
}

// beginUpdate marks the start of one or more updates to the application
// screen.
func (a *Application) beginUpdate() {
	if atomic.AddInt32(&a.updateLevel, 1) == 1 {
		a.paintSelection() // Remove selection.
	}
}

// endUpdate marks the end of one or more updates to the application screen.
func (a *Application) endUpdate() {
	if atomic.AddInt32(&a.updateLevel, -1) == 0 {
		a.paintSelection() // Show selection.
		a.screen.Show()
	}
}

func (a *Application) setCell(x, y int, mainc rune, combc []rune, style tcell.Style) {
	a.screen.SetContent(x, y, mainc, combc, style)
}

// ----------------------------------------------------------------------------

// ChildWindowStyle returns the style assigned to new child windows.
func (a *Application) ChildWindowStyle() WindowStyle { return getWindowStyle(nil, &a.theme.ChildWindow) }

// ClickDuration returns the maximum duration of a single click. Holding a
// mouse button for any longer duration generates a drag event instead.
func (a *Application) ClickDuration() time.Duration { return getDuration(nil, &a.click) }

// Dektop returns the currently active desktop.
func (a *Application) Desktop() (d *Desktop) { return getDesktop(nil, &a.desktop) }

// DektopStyle returns the style assigned to new desktops.
func (a *Application) DesktopStyle() WindowStyle { return getWindowStyle(nil, &a.theme.Desktop) }

// DoubleClickDuration returns the maximum duration of a double click. Mouse
// click not followed by another one within the DoubleClickDuration is a single
// click.
func (a *Application) DoubleClickDuration() time.Duration { return getDuration(nil, &a.doubleClick) }

// Exit terminates the interactive terminal application and returns err from
// Wait(). Calling this method more than once will panic.
func (a *Application) Exit(err error) {
	a.mu.Lock()
	if a.terminated {
		panic("Application.Exit called more than once")
	}

	a.Finalize()
	a.terminated = true
	a.wait <- err
	a.mu.Unlock()
}

// Finalize should be called when main exits to restore the normal terminal
// state.
//
// Calling this method more than once will panic.
func (a *Application) Finalize() {
	if a.terminated {
		return
	}

	done := false
	a.onceFinalize.Do(func() {
		a.screen.Fini()
		done = true
	})
	if !done {
		panic("Application.Finalize called more than once")
	}
}

// NewDesktop returns a newly created desktop.
func (a *Application) NewDesktop() *Desktop { return newDesktop() }

// OnKey sets a key event handler. When the event handler is removed, finalize
// is called, if not nil.
func (a *Application) OnKey(h OnKeyHandler, finalize func()) {
	addOnKeyHandler(nil, &a.onKey, h, finalize)
}

// OnSetClickDuration sets a handler invoked on SetClickDuration. When the
// event handler is removed, finalize is called, if not nil.
func (a *Application) OnSetClickDuration(h OnSetDurationHandler, finalize func()) {
	addOnSetDurationHandler(nil, &a.onSetClick, h, finalize)
}

// OnSetDesktop sets a handler invoked on SetDesktop. When the event handler is
// removed, finalize is called, if not nil.
func (a *Application) OnSetDesktop(h OnSetDesktopHandler, finalize func()) {
	addOnSetDesktopHandler(nil, &a.onSetDesktop, h, finalize)
}

// OnSetDoubleClickDuration sets a handler invoked on SetDoubleClickDuration.
// When the event handler is removed, finalize is called, if not nil.
func (a *Application) OnSetDoubleClickDuration(h OnSetDurationHandler, finalize func()) {
	addOnSetDurationHandler(nil, &a.onSetDoubleClick, h, finalize)
}

// OnSetSize sets a handler invoked on resizing the application screen. When
// the event handler is removed, finalize is called, if not nil.
func (a *Application) OnSetSize(h OnSetSizeHandler, finalize func()) {
	addOnSetSizeHandler(nil, &a.onSetSize, h, finalize)
}

// Post puts f in the event queue, if the queue is not full, and executes it on
// dequeuing the event.
func (a *Application) Post(f func()) { a.screen.PostEvent(newEventFunc(f)) }

// PostWait puts f in the event queue and executes it on dequeuing the event.
func (a *Application) PostWait(f func()) { a.screen.PostEventWait(newEventFunc(f)) }

// RemoveOnKey undoes the most recent OnKey call. The function will panic if
// there is no handler set.
func (a *Application) RemoveOnKey() { removeOnKeyHandler(nil, &a.onKey) }

// RemoveOnSetClickDuration undoes the most recent OnSetClickDuration call. The
// function will panic if there is no handler set.
func (a *Application) RemoveOnSetClickDuration() { removeOnSetDurationHandler(nil, &a.onSetClick) }

// RemoveOnSetDesktop undoes the most recent OnSetDesktop call. The function
// will panic if there is no handler set.
func (a *Application) RemoveOnSetDesktop() { removeOnSetDesktopHandler(nil, &a.onSetDesktop) }

// RemoveOnSetDoubleClickDuration undoes the most recent
// OnSetDoubleClickDuration call. The function will panic if there is no
// handler set.
func (a *Application) RemoveOnSetDoubleClickDuration() {
	removeOnSetDurationHandler(nil, &a.onSetDoubleClick)
}

// RemoveOnSetSize undoes the most recent OnSetSize call. The function
// will panic if there is no handler set.
func (a *Application) RemoveOnSetSize() { removeOnSetSizeHandler(nil, &a.onSetSize) }

// SetClickDuration sets the maximum duration of a single click. Holding a
// mouse button for any longer duration generates a drag event instead.
func (a *Application) SetClickDuration(d time.Duration) { a.onSetClick.handle(nil, &a.click, d) }

// SetDesktop sets the currently active desktop. Passing nil d will panic.
func (a *Application) SetDesktop(d *Desktop) {
	if d == nil {
		panic("cannot set nil desktop")
	}

	a.onSetDesktop.handle(nil, &a.desktop, d)
}

// SetDoubleClickDuration sets the maximum duration of a single click. Mouse
// click not followed by another one within the DoubleClickDuration is a single
// click.
//
// Note: Setting DoubleClickDuration to zero disables double click support.
func (a *Application) SetDoubleClickDuration(d time.Duration) {
	a.onSetClick.handle(nil, &a.doubleClick, d)
}

func (a *Application) setSize(s Size) { a.onSetSize.handle(nil, &a.size, s) }

// Size returns the size of the terminal the application runs in.
func (a *Application) Size() (s Size) { return getSize(nil, &a.size) }

// Sync updates every character cell of the application screen.
func (a *Application) Sync() { a.screen.Sync() }

// Wait blocks until the interactive terminal application terminates.
//
// Calling this method more than once will panic.
func (a *Application) Wait() error {
	done := false
	var err error
	a.onceWait.Do(func() {
		err = <-a.wait
		done = true
	})
	if !done {
		panic("Application.Wait called more than once")
	}

	return err
}
