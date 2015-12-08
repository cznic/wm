// Copyright 2015 The WM Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wm

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/gdamore/tcell"
)

var (
	zeroStyle Style
)

// Style represents a text style.
type Style struct {
	Foreground tcell.Color
	Background tcell.Color
	Attr       tcell.AttrMask
}

// IsZero returns whether s is the zero value of Style.
func (s *Style) IsZero() bool { return *s == zeroStyle }

// NewStyle returns Style having values filled from s.
func NewStyle(s tcell.Style) Style {
	f, b, a := s.Decompose()
	return Style{f, b, a}
}

// TCellStyle converts a Style to a tcell.Style value.
func (s Style) TCellStyle() tcell.Style {
	return tcell.Style(0).
		Foreground(s.Foreground).
		Background(s.Background).
		Bold(s.Attr&tcell.AttrBold != 0).
		Blink(s.Attr&tcell.AttrBlink != 0).
		Reverse(s.Attr&tcell.AttrReverse != 0).
		Underline(s.Attr&tcell.AttrUnderline != 0).
		Dim(s.Attr&tcell.AttrDim != 0)

}

// Theme represents visual styles of UI elements.
type Theme struct {
	ChildWindow WindowStyle
	Desktop     WindowStyle
}

// WindowStyle represents visual styles of a Window.
type WindowStyle struct {
	Border     Style
	ClientArea Style
	Title      Style
}

// Clear sets t to its zero value.
func (t *Theme) Clear() { *t = Theme{} }

// WriteTo writes t to w in JSON format.
func (t *Theme) WriteTo(w io.Writer) error {
	b, err := json.Marshal(t)
	if err != nil {
		return err
	}

	_, err = w.Write(b)
	return err
}

// ReadFrom reads t from r in JSON format. Values of fields having no JSON data
// are preserved.
func (t *Theme) ReadFrom(r io.Reader) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, t)
}
