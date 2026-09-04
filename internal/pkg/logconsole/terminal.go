// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package logconsole

import (
	"io"
	"os"
)

// IsTerminal reports whether w is an interactive terminal.
func IsTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
