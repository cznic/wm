// Copyright 2015 The WM Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

// $ go run demo.go
package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/cznic/wm"
	"github.com/cznic/wm/internal/demoapp"
	"github.com/gdamore/tcell"
	"github.com/golang/glog"
)

const (
	block = tcell.RuneBlock
)

var (
	theme = &wm.Theme{
		ChildWindow: wm.WindowStyle{
			Border:     wm.Style{Background: tcell.ColorFuchsia, Foreground: tcell.ColorGreen},
			ClientArea: wm.Style{Background: tcell.ColorSilver, Foreground: tcell.ColorYellow},
			Title:      wm.Style{Background: tcell.ColorGreen, Foreground: tcell.ColorRed},
		},
		Desktop: wm.WindowStyle{
			ClientArea: wm.Style{Background: tcell.ColorBlue, Foreground: tcell.ColorWhite},
		},
	}
	mouseMoveWindow *wm.Window
	mousePos        wm.Position
)

func rndColor() tcell.Color {
	for {
		c := tcell.Color(rand.Intn(wm.App.Colors()))
		if c != demoapp.Theme.ChildWindow.ClientArea.Background {
			return c
		}
	}
}

func newWindow(parent *wm.Window, x, y int) {
	a := parent.Size()
	if x < 0 || y < 0 {
		x = rand.Intn(a.Width - a.Width/5)
		y = rand.Intn(a.Height - a.Height/5)
	}
	w := rand.Intn(a.Width/2) + 10
	h := rand.Intn(a.Height/2) + 10
	c := parent.NewChild(wm.Rectangle{wm.Position{x, y}, wm.Size{w, h}})
	style := tcell.Style(0).Foreground(rndColor())
	c.SetCloseButton(true)
	c.SetTitle(time.Now().Format("15:04:05"))
	c.OnPaintClientArea(
		func(w *wm.Window, prev wm.OnPaintHandler, ctx wm.PaintContext) {
			if prev != nil {
				prev(w, nil, ctx)
			}

			w.Printf(0, 0, w.ClientAreaStyle(), "abc %v %v %v", c.Position(), c.Size(), w.ClientArea())
			w.Printf(0, 1, w.ClientAreaStyle(), "def %v %v %v", c.Position(), c.Size(), w.ClientArea())
			w.Printf(0, 2, w.ClientAreaStyle(), "ghi %v %v %v", c.Position(), c.Size(), w.ClientArea())
			w.SetCell(c.ClientArea().Size.Width/2, c.ClientArea().Size.Height/2, block, nil, style)
		},
		nil,
	)
	c.OnClick(
		func(w *wm.Window, prev wm.OnMouseHandler, button tcell.ButtonMask, screenPos, winPos wm.Position, mods tcell.ModMask) bool {
			if prev != nil && prev(w, nil, button, screenPos, winPos, mods) {
				return true
			}

			switch {
			case mods&tcell.ModCtrl != 0:
				newWindow(w, winPos.X, winPos.Y)
				return true
			default:
				style = tcell.Style(0).Foreground(rndColor())
				x, y = winPos.X, winPos.Y
				return true
			}
		},
		nil,
	)
	c.OnMouseMove(onMouseMove, nil)
	c.SetFocus(true)
}

func onMouseMove(w *wm.Window, prev wm.OnMouseHandler, button tcell.ButtonMask, screenPos, winPos wm.Position, mods tcell.ModMask) bool {
	mousePos = winPos
	mouseMoveWindow = w
	if prev != nil {
		return prev(w, nil, button, screenPos, winPos, mods)
	}

	return true
}

func setup(d *wm.Desktop) {
	app := wm.App
	app.SetDoubleClickDuration(0)
	r := d.Root()
	r.OnPaintClientArea(
		func(w *wm.Window, prev wm.OnPaintHandler, ctx wm.PaintContext) {
			if prev != nil {
				prev(w, nil, ctx)
			}

			mousePosStr := ""
			if w == mouseMoveWindow {
				mousePosStr = fmt.Sprintf("Mouse: %+v", mousePos)
			}
			w.Printf(0, 0, w.ClientAreaStyle(),
				`Ctrl-N to create a new random window.
Ctrl-click inside a child window to create a nested random window.
Use mouse to bring to front, drag, resize or close a window.
Arrow keys change the viewport of the focused window.
To focus the desktop, click on it.
<Esc> or <q> to quit.
Rendered in %s.
%s`, r.Rendered(), mousePosStr)
		},
		nil,
	)
	r.OnMouseMove(onMouseMove, nil)
	app.OnKey(
		func(w *wm.Window, prev wm.OnKeyHandler, key tcell.Key, mod tcell.ModMask, r rune) bool {
			switch r {
			case 'q', 'Q':
				app.Exit(nil)
				return true
			}

			switch key {
			case tcell.KeyESC:
				app.Exit(nil)
				return true
			case tcell.KeyCtrlQ:
				app.Exit(nil)
				return true
			case tcell.KeyCtrlN:
				glog.Info("==== CTRL-N")
				newWindow(app.Desktop().Root(), -1, -1)
				return true
			case tcell.KeyLeft:
				w := d.FocusedWindow()
				if w == nil {
					return true
				}

				o := w.Origin()
				w.SetOrigin(wm.Position{X: o.X - 1, Y: o.Y})
				return true
			case tcell.KeyRight:
				w := d.FocusedWindow()
				if w == nil {
					return true
				}

				o := w.Origin()
				w.SetOrigin(wm.Position{X: o.X + 1, Y: o.Y})
				return true
			case tcell.KeyUp:
				w := d.FocusedWindow()
				if w == nil {
					return true
				}

				o := w.Origin()
				w.SetOrigin(wm.Position{X: o.X, Y: o.Y - 1})
				return true
			case tcell.KeyDown:
				w := d.FocusedWindow()
				if w == nil {
					return true
				}

				o := w.Origin()
				w.SetOrigin(wm.Position{X: o.X, Y: o.Y + 1})
				return true
			case tcell.KeyF2:
				w := d.FocusedWindow()
				if w == nil {
					return true
				}

				w.Invalidate(wm.NewRectangle(1, 1, 11, 11))
				return true
			case tcell.KeyF3:
				w := d.FocusedWindow()
				if w == nil {
					return true
				}

				w.InvalidateClientArea(wm.NewRectangle(1, 1, 11, 11))
				return true
			default:
				return false
			}
		},
		nil,
	)
	d.Show()
}

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	app, d := demoapp.New()
	if err := app.Run(func() { setup(d) }); err != nil {
		log.Fatal(err)
	}

	glog.Flush()
}
