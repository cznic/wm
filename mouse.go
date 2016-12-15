// Copyright 2015 The WM Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wm

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell"
)

type mbState int

const (
	mbsIdle mbState = iota
	mbsDown
	mbsUp
	mbsDown2
	mbsDrag
)

type mouseButtonFSM struct {
	in      chan *tcell.EventMouse //
	button  tcell.ButtonMask       //
	mods    tcell.ModMask          //
	pos     Position               //
	quit    chan struct{}          //
	state   mbState                //
	timeout <-chan time.Time       //
}

func newMouseButtonFSM(button tcell.ButtonMask) *mouseButtonFSM {
	m := &mouseButtonFSM{
		in:     make(chan *tcell.EventMouse, 1),
		button: button,
		quit:   make(chan struct{}, 1),
	}
	go m.run()
	return m
}

func (m *mouseButtonFSM) post(e *tcell.EventMouse) { m.in <- e }

func (m *mouseButtonFSM) close() {
	select {
	case m.quit <- struct{}{}:
	default:
	}
}

func (m *mouseButtonFSM) run() {
	for {
		switch m.state {
		case mbsIdle:
			select {
			case e := <-m.in:
				switch e.Buttons() & m.button {
				case 0: // Button up.
					// nop
				default: // Button down.
					m.mods = e.Modifiers()
					x, y := e.Position()
					m.pos = Position{x, y}
					m.timeout = time.After(App.ClickDuration())
					m.state = mbsDown
				}
			case <-m.timeout:
				m.timeout = nil
			case <-m.quit:
				return
			}
		case mbsDown:
			select {
			case e := <-m.in:
				switch e.Buttons() & m.button {
				case 0: // Button up.
					if d := App.DoubleClickDuration(); d != 0 {
						m.timeout = time.After(d)
						m.state = mbsUp
						break
					}

					App.screen.PostEvent(newEventMouse(mouseClick, m.button, m.mods, m.pos))
					m.state = mbsIdle
					m.timeout = nil
				default: // Button down.
					m.state = mbsIdle
				}
			case <-m.timeout:
				App.screen.PostEvent(newEventMouse(mouseDrag, m.button, m.mods, m.pos))
				m.state = mbsDrag
			case <-m.quit:
				return
			}
		case mbsUp:
			select {
			case e := <-m.in:
				switch e.Buttons() & m.button {
				case 0: // Button up.
					m.state = mbsIdle
				default: // Button down.
					App.screen.PostEvent(newEventMouse(mouseDoubleClick, m.button, m.mods, m.pos))
					m.state = mbsDown2
				}
			case <-m.timeout:
				App.screen.PostEvent(newEventMouse(mouseClick, m.button, m.mods, m.pos))
				m.state = mbsIdle
				m.timeout = nil
			case <-m.quit:
				return
			}
		case mbsDown2:
			select {
			case e := <-m.in:
				switch e.Buttons() & m.button {
				case 0: // Button up.
					m.state = mbsIdle
				default: // Button down.
					m.state = mbsIdle
				}
			case <-m.timeout:
				m.timeout = nil
			case <-m.quit:
				return
			}
		case mbsDrag:
			select {
			case e := <-m.in:
				switch e.Buttons() & m.button {
				case 0: // Button up.
					x, y := e.Position()
					App.screen.PostEvent(newEventMouse(mouseDrop, m.button, e.Modifiers(), Position{x, y}))
					m.state = mbsIdle
					m.timeout = nil
				default: // Button down.
					m.state = mbsIdle
				}
			case <-m.timeout:
				m.timeout = nil
			case <-m.quit:
				return
			}
		default:
			panic(fmt.Errorf("%v", m.state))
		}
	}
}
