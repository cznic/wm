// Copyright 2015 The WM Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package wm is a terminal window manager.
package wm

import (
	"sync"
)

var (
	app                *Application
	onceNewApplication sync.Once
)
