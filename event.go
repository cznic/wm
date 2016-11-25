// Copyright 2015 The WM Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wm

import (
	"sync"
	"time"

	"github.com/gdamore/tcell"
)

var (
	_ tcell.Event = (*eventMouse)(nil)
	_ tcell.Event = (*eventFunc)(nil)
)

var (
	eventFuncPool  = sync.Pool{New: func() interface{} { return &eventFunc{} }}
	eventMousePool = sync.Pool{New: func() interface{} { return &eventMouse{} }}
)

type event struct{}

func (e event) When() time.Time { return time.Time{} }

const (
	_ = iota //TODOOK
	mouseClick
	mouseDoubleClick
	mouseDrag
	mouseDrop
	mouseMove
)

type eventMouse struct {
	Position
	button tcell.ButtonMask
	event
	kind int
	mods tcell.ModMask
}

func newEventMouse(kind int, button tcell.ButtonMask, mods tcell.ModMask, pos Position) *eventMouse {
	e := eventMousePool.Get().(*eventMouse)
	e.Position = pos
	e.button = button
	e.kind = kind
	e.mods = mods
	return e
}

func (e *eventMouse) dispose() {
	*e = eventMouse{}
	eventMousePool.Put(e)
}

type eventFunc struct {
	event
	f func()
}

func newEventFunc(f func()) *eventFunc {
	e := eventFuncPool.Get().(*eventFunc)
	e.f = f
	return e
}

func (e *eventFunc) dispose() {
	*e = eventFunc{}
	eventFuncPool.Put(e)
}
