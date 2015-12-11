// Copyright 2015 The WM Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

// $ go run demo.go
package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/cznic/wm"
	"github.com/gdamore/tcell"
)

const (
	block = tcell.RuneBlock
)

var (
	app   *wm.Application
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
		c := tcell.Color(rand.Intn(app.Colors()))
		if c != theme.ChildWindow.ClientArea.Background {
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
	x, y = 0, 0
	dx, dy := 1, 1
	t := time.NewTicker(time.Millisecond * time.Duration(35+rand.Intn(10)))
	c.SetCloseButton(true)
	c.SetTitle(time.Now().Format("15:04:05"))
	style := tcell.Style(0).Foreground(rndColor())
	c.OnPaintClientArea(
		func(w *wm.Window, prev wm.OnPaintHandler, ctx wm.PaintContext) {
			if prev != nil {
				prev(w, nil, ctx)
			}

			select {
			case <-t.C:
				a := w.ClientArea()
				if x >= a.Width-1 {
					if x > a.Width {
						x = a.Width
					}
					dx = -1
				} else if x <= 0 {
					if x < 0 {
						x = -1
					}
					dx = 1
				}
				if y >= a.Height-1 {
					if y > a.Height {
						y = a.Height
					}
					dy = -1
				} else if y <= 0 {
					if y < 0 {
						y = -1
					}
					dy = 1
				}

				x += dx
				y += dy
			default:
			}
			if w == mouseMoveWindow {
				w.Printf(0, 0, w.ClientAreaStyle(), "Mouse: %+v", mousePos)
			}
			w.SetCell(x, y, block, nil, style)
		},
		nil,
	)
	c.OnClose(
		func(w *wm.Window, prev wm.OnCloseHandler) {
			if prev != nil {
				prev(w, nil)
			}
			t.Stop()
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

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile | log.Ltime)

	rand.Seed(time.Now().UnixNano())

	var err error
	if app, err = wm.NewApplication(theme); err != nil {
		log.Fatal(err)
	}

	defer app.Finalize()

	app.SetDoubleClickDuration(0)
	d := app.NewDesktop()
	r := d.Root()
	var renderedIn time.Duration
	r.OnPaintClientArea(
		func(w *wm.Window, prev wm.OnPaintHandler, ctx wm.PaintContext) {
			if prev != nil {
				prev(w, nil, ctx)
			}

			w.Printf(0, 0, w.ClientAreaStyle(),
				`Ctrl-N to create a new random window.
Ctrl-click inside a child window to create a nested random window.
Use mouse to bring to front, drag, resize or close a window.
Esc to quit.
Rendered in %s.`, renderedIn)
			if w == mouseMoveWindow {
				w.Printf(0, 5, w.ClientAreaStyle(), "Mouse: %+v", mousePos)
			}
		},
		nil,
	)
	r.OnMouseMove(onMouseMove, nil)
	app.OnKey(
		func(w *wm.Window, prev wm.OnKeyHandler, key tcell.Key, mod tcell.ModMask, r rune) bool {
			switch key {
			case tcell.KeyESC:
				app.Exit(nil)
				return true
			case tcell.KeyCtrlN:
				newWindow(app.Desktop().Root(), -1, -1)
				return true
			default:
				return false
			}
		},
		nil,
	)
	d.Show()
	fps := 25
	i := 0
	go func() {
		for range time.NewTicker(time.Second / time.Duration(fps)).C {
			app.Post(func() {
				i = (i + 1) % fps
				var t0 time.Time
				if i == 0 {
					t0 = time.Now()
				}
				r.Invalidate(r.Area())
				if i == 0 {
					renderedIn = time.Since(t0)
				}
			})
		}
	}()

	if err := app.Wait(); err != nil {
		log.Println(err)
	}
}
