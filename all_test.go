// Copyright 2015 The WM Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wm

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"

	"github.com/gdamore/tcell"
)

func caller(s string, va ...interface{}) {
	if s == "" {
		s = strings.Repeat("%v ", len(va))
	}
	_, fn, fl, _ := runtime.Caller(2)
	fmt.Fprintf(os.Stderr, "// caller: %s:%d: ", path.Base(fn), fl)
	fmt.Fprintf(os.Stderr, s, va...)
	fmt.Fprintln(os.Stderr)
	_, fn, fl, _ = runtime.Caller(1)
	fmt.Fprintf(os.Stderr, "// \tcallee: %s:%d: ", path.Base(fn), fl)
	fmt.Fprintln(os.Stderr)
	os.Stderr.Sync()
}

func dbg(s string, va ...interface{}) {
	if s == "" {
		s = strings.Repeat("%v ", len(va))
	}
	_, fn, fl, _ := runtime.Caller(1)
	fmt.Fprintf(os.Stderr, "// dbg %s:%d: ", path.Base(fn), fl)
	fmt.Fprintf(os.Stderr, s, va...)
	fmt.Fprintln(os.Stderr)
	os.Stderr.Sync()
}

func TODO(...interface{}) string { //TODOOK
	_, fn, fl, _ := runtime.Caller(1)
	return fmt.Sprintf("// TODO: %s:%d:\n", path.Base(fn), fl) //TODOOK
}

func use(...interface{}) {}

func init() {
	use(caller, dbg, TODO) //TODOOK
}

// ============================================================================

func TestJoin(t *testing.T) {
	a := Rectangle{Position{X: 46, Y: 12}, Size{Width: 55, Height: 27}}
	b := Rectangle{Position{X: 45, Y: 12}, Size{Width: 55, Height: 27}}
	a.join(b)
	if g, e := a, (Rectangle{Position{X: 45, Y: 12}, Size{Width: 56, Height: 27}}); g != e {
		t.Fatal(g, e)
	}
}

func TestDesktopPaintContext(t *testing.T) {
	s := tcell.NewSimulationScreen("")
	app, err := newApplication(s, &Theme{})
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		app.PostWait(func() { app.Exit(nil) })
		if err := app.Wait(); err != nil {
			t.Fatal(err)
		}
	}()

	var r *Window
	var c, cc PaintContext
	ch := make(chan int, 1)
	app.PostWait(func() {
		d := app.NewDesktop()
		r = d.Root()
		app.SetDesktop(d)
		AddOnPaintHandler(&r.onClearClientArea, func(w *Window, prev OnPaintHandler, ctx PaintContext) {
			if prev != nil {
				prev(w, nil, ctx)
			}
			cc = ctx
		}, nil)
		r.OnPaintClientArea(func(w *Window, prev OnPaintHandler, ctx PaintContext) {
			if prev != nil {
				prev(w, nil, ctx)
			}
			c = ctx
			ch <- 1
		}, nil)
		d.Show()
	})

	app.PostWait(func() {
		r.InvalidateClientArea(r.ClientArea())
	})
	<-ch
	if g, e := cc, (PaintContext{
		Rectangle: Rectangle{Position{}, Size{80, 25}},
		origin:    Position{},
		view:      Position{},
	}); g != e {
		t.Fatalf("\n%+v\n%+v", g, e)
	}
	if g, e := c, (PaintContext{
		Rectangle: Rectangle{Position{}, Size{80, 25}},
		origin:    Position{},
		view:      Position{},
	}); g != e {
		t.Fatalf("\n%+v\n%+v", g, e)
	}

	app.PostWait(func() {
		r.SetOrigin(Position{2, 1})
		r.InvalidateClientArea(r.ClientArea())
	})
	<-ch
	if g, e := cc, (PaintContext{
		Rectangle: Rectangle{Position{}, Size{80, 25}},
		origin:    Position{},
		view:      Position{},
	}); g != e {
		t.Fatalf("\n%+v\n%+v", g, e)
	}
	if g, e := c, (PaintContext{
		Rectangle: Rectangle{Position{2, 1}, Size{80, 25}},
		origin:    Position{},
		view:      Position{2, 1},
	}); g != e {
		t.Fatalf("\n%+v\n%+v", g, e)
	}
}
