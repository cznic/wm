// Copyright 2016 The WM Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

// $ go run view_demo.go
package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	"github.com/cznic/mathutil"
	"github.com/cznic/wm"
	"github.com/cznic/wm/internal/demoapp"
	"github.com/cznic/wm/tk"
	"github.com/gdamore/tcell"
)

const (
	help = `Use mouse to resize the window or scroll the view.
Arrow keys change the viewport of the focused window.
To focus the desktop, click on it.
<Esc> or 'q' to quit.`
)

var nl = []byte{'\n'}

func newWindow(parent *wm.Window, x, y int, title string, src []byte) {
	sz := parent.Size()
	if x < 0 || y < 0 {
		x = rand.Intn(sz.Width - sz.Width/5)
		y = rand.Intn(sz.Height - sz.Height/5)
	}
	c := parent.NewChild(wm.Rectangle{wm.Position{x, y}, wm.Size{0, 0}})
	c.SetCloseButton(true)
	c.SetTitle(title)
	if bytes.HasSuffix(src, nl) {
		src = src[:len(src)-1]
	}
	a := bytes.Split([]byte(src), nl)
	max := -1
	for _, v := range a {
		var x int
		for _, c := range v {
			switch c {
			case '\t':
				x += 8 - x%8
			default:
				x++
			}
		}
		max = mathutil.Max(max, x)
	}
	c.OnPaintClientArea(
		func(w *wm.Window, prev wm.OnPaintHandler, ctx wm.PaintContext) {
			if prev != nil {
				prev(w, nil, ctx)
			}
			cpY := w.ClientPosition().Y
			for i := 0; i < ctx.Height; i++ {
				line := ctx.Y - cpY + i
				if line >= len(a) {
					break
				}

				w.Printf(0, line, w.ClientAreaStyle(), "%s", a[line])
			}
		},
		nil,
	)
	v := tk.NewView(c, meter{max, len(a)})
	c.OnKey(
		func(w *wm.Window, prev wm.OnKeyHandler, key tcell.Key, mod tcell.ModMask, r rune) bool {
			if prev != nil && prev(w, nil, key, mod, r) {
				return true
			}

			switch key {
			case tcell.KeyHome:
				v.Home()
				return true
			case tcell.KeyEnd:
				v.End()
				return true
			case tcell.KeyPgDn:
				v.PageDown()
				return true
			case tcell.KeyPgUp:
				v.PageUp()
				return true
			}
			return false
		},
		nil,
	)
	c.SetSize(wm.Size{Width: rand.Intn(sz.Width/2) + 20, Height: rand.Intn(sz.Height/2) + 15})
	c.SetFocus(true)
}

func setup(d *wm.Desktop) {
	app := wm.App
	app.SetDoubleClickDuration(0)

	defer d.Show()

	r := d.Root()
	r.OnPaintClientArea(
		func(w *wm.Window, prev wm.OnPaintHandler, ctx wm.PaintContext) {
			if prev != nil {
				prev(w, nil, ctx)
			}

			w.Printf(0, 0, w.ClientAreaStyle(), "%s", help)
		}, nil,
	)
	app.OnKey(
		func(w *wm.Window, prev wm.OnKeyHandler, key tcell.Key, mod tcell.ModMask, rune rune) bool {
			if prev != nil && prev(w, nil, key, mod, rune) {
				return true
			}

			switch rune {
			case 'q', 'Q':
				app.Exit(nil)
				return true
			}

			switch key {
			case tcell.KeyCtrlQ:
				app.Exit(nil)
				return true
			case tcell.KeyTAB:
				if n := r.Children(); n != 0 {
					c := r.Child(0)
					c.BringToFront()
					c.SetFocus(true)
					return true
				}
			case tcell.KeyESC:
				app.Exit(nil)
				return true
			case tcell.KeyLeft:
				w := d.FocusedWindow()
				if w == nil {
					return true
				}

				o := w.Origin()
				w.SetOrigin(wm.Position{X: o.X - 1, Y: o.Y})
				r.Invalidate(r.Area())
				return true
			case tcell.KeyRight:
				w := d.FocusedWindow()
				if w == nil {
					return true
				}

				o := w.Origin()
				w.SetOrigin(wm.Position{X: o.X + 1, Y: o.Y})
				r.Invalidate(r.Area())
				return true
			case tcell.KeyUp:
				w := d.FocusedWindow()
				if w == nil {
					return true
				}

				o := w.Origin()
				w.SetOrigin(wm.Position{X: o.X, Y: o.Y - 1})
				r.Invalidate(r.Area())
				return true
			case tcell.KeyDown:
				w := d.FocusedWindow()
				if w == nil {
					return true
				}

				o := w.Origin()
				w.SetOrigin(wm.Position{X: o.X, Y: o.Y + 1})
				r.Invalidate(r.Area())
				return true
			}
			return false
		},
		nil,
	)
	m, err := filepath.Glob("*")
	if err == nil {
		for _, v := range m {
			if strings.HasPrefix(v, ".") {
				continue
			}

			b, err := ioutil.ReadFile(v)
			if err != nil {
				continue
			}

			newWindow(r, -1, -1, v, b)
		}
	}
}

type meter wm.Size

func (m meter) Metrics(_ wm.Rectangle) wm.Size { return wm.Size(m) }

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	app, d := demoapp.New()
	if err := app.Run(func() { setup(d) }); err != nil {
		log.Fatal(err)
	}
}
