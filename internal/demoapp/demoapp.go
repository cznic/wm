// Copyright 2016 The WM Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package demoapp is a terminal application skeleton.
package demoapp

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/cznic/wm"
	"github.com/gdamore/tcell"
)

var (
	windowStyle = wm.WindowStyle{
		Border:     wm.Style{Background: tcell.ColorNavy, Foreground: tcell.ColorGreen},
		ClientArea: wm.Style{Background: tcell.ColorSilver, Foreground: tcell.ColorNavy},
		Title:      wm.Style{Background: tcell.ColorNavy, Foreground: tcell.ColorSilver},
	}

	Theme = &wm.Theme{
		ChildWindow: windowStyle,
		Desktop: wm.WindowStyle{
			ClientArea: wm.Style{Background: tcell.ColorTeal, Foreground: tcell.ColorWhite},
		},
	}

	logoStyle  = wm.Style{Background: Theme.Desktop.ClientArea.Background, Foreground: tcell.ColorWhite}
	pnameStyle = wm.Style{Background: Theme.Desktop.ClientArea.Background, Foreground: tcell.ColorNavy}

	oLog = flag.String("log", "", "log file")
)

// New returns a newly created terminal application and a newly created desktop
// which is not yet shown.
//
// Double click events are disabled to improve single click response time. To
// enable double click events use wm.App.SetDoubleClickDuration to a nonzero
// value.
func New() (*wm.Application, *wm.Desktop) {
	logf := *oLog
	if logf == "" {
		logf = os.DevNull
	}
	var f *os.File
	fi, err := os.Stat(logf)
	switch {
	case err == nil:
		if fi.Mode()&os.ModeNamedPipe != 0 {
			if f, err = os.OpenFile(logf, os.O_WRONLY, 0666); err != nil {
				log.Fatal(err)
			}

			break
		}

		fallthrough
	default:
		if f, err = os.Create(logf); err != nil {
			log.Fatal(err)
		}
	}

	log.SetOutput(f)
	app, err := wm.NewApplication(Theme)
	if err != nil {
		log.Fatal(err)
	}

	const (
		logo   = "github.com/cznic/wm"
		border = 1
	)
	pname := os.Args[0]
	if fi, err = os.Stat(pname); err == nil {
		pname = fmt.Sprintf("%s %s", pname, fi.ModTime().Format("2006-01-02 15:04:05"))
	}
	d := app.NewDesktop()
	app.PostWait(func() {
		app.SetDoubleClickDuration(0)
		d := app.NewDesktop()
		d.Root().OnPaintClientArea(func(w *wm.Window, prev wm.OnPaintHandler, ctx wm.PaintContext) {
			if prev != nil {
				prev(w, nil, ctx)
			}

			sz := w.Size()
			w.Printf(sz.Width-border-len(pname), 0, pnameStyle, "%s", pname)
			w.Printf(sz.Width-border-len(logo), sz.Height-border-1, logoStyle, "%s", logo)
		}, nil)
	})
	return app, d
}
