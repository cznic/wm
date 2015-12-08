// Copyright 2015 The WM Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wm

import (
	"sync"
)

// Desktop represents a virtual screen. An application has one or more
// independent desktops, of which only one is visible at any given moment.
//
// A desktop initially contains only the automatically created root window.
//
// All exported methods of a Desktop are safe for concurrent access by multiple
// goroutines.
type Desktop struct {
	invalidated Rectangle  //
	mu          sync.Mutex //
	root        *Window    // Never changes.
	updateLevel int        //
}

func newDesktop() *Desktop {
	d := &Desktop{}
	w := newWindow(d, nil, app.DesktopStyle())
	d.root = w
	w.setSize(app.Size())
	d.OnSetSelection(w.onSetSelectionHandler, nil)
	d.OnSetFocusedWindow(w.onSetFocusedWindowHandler, nil)
	return d
}

// ----------------------------------------------------------------------------

// FocusedWindow returns the window with focus, if any.
func (d *Desktop) FocusedWindow() *Window {
	r := d.root
	if r == nil {
		return nil
	}

	return getWindow(r, &r.focusedWindow)
}

// OnSetFocusedWindow sets a handler invoked on SetFocusedWindow. When the
// event handler is removed, finalize is called, if not nil.
func (d *Desktop) OnSetFocusedWindow(h OnSetWindowHandler, finalize func()) {
	r := d.Root()
	if r == nil {
		return
	}

	addOnSetWindowHandler(r, &r.onSetFocusedWindow, h, finalize)
}

// OnSetSelection sets a handler invoked on SetSelection. When the event
// handler is removed, finalize is called, if not nil.
func (d *Desktop) OnSetSelection(h OnSetRectangleHandler, finalize func()) {
	r := d.Root()
	if r == nil {
		return
	}

	addOnSetRectangleHandler(r, &r.onSetSelection, h, finalize)
}

// RemoveOnSetFocusedWindow undoes the most recent OnSetFocusedWindow call. The
// function will panic if there is no handler set.
func (d *Desktop) RemoveOnSetFocusedWindow() {
	r := d.Root()
	if r == nil {
		return
	}

	removeOnSetWindowHandler(r, &r.onSetFocusedWindow)
}

// RemoveOnSetSelection undoes the most recent OnSetSelection call. The
// function will panic if there is no handler set.
func (d *Desktop) RemoveOnSetSelection() {
	r := d.Root()
	if r == nil {
		return
	}

	removeOnSetRectangleHandler(r, &r.onSetSelection)
}

// Root returns the root window of d.
func (d *Desktop) Root() *Window { return d.root }

// Selection returns the area of the desktop shown in reverse.
func (d *Desktop) Selection() Rectangle {
	r := d.Root()
	if r == nil {
		return Rectangle{}
	}

	return getRectangle(r, &r.selection)
}

// SetFocusedWindow sets the focused window.
func (d *Desktop) SetFocusedWindow(w *Window) {
	r := d.root
	if r == nil {
		return
	}

	r.setFocusedWindow(w)
}

// SetSelection sets the area of the desktop shown in reverse.
func (d *Desktop) SetSelection(area Rectangle) {
	r := d.Root()
	if r == nil {
		return
	}

	r.onSetSelection.handle(r, &r.selection, area)
}

// Show sets d as the application active desktop.
func (d *Desktop) Show() { app.SetDesktop(d) }
