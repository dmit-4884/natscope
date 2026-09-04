// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package appconfig

import (
	"github.com/altessa-s/go-atlas/config"

	"github.com/dmit-4884/natscope/internal/pkg/logconsole"
)

// LoggerDefaults returns the logger defaults seeded before the config file and environment apply.
func LoggerDefaults(interactive bool) *config.Logger {
	l := &config.Logger{
		Level:        config.LoggerLevelInfo,
		Output:       config.LoggerConsoleOutputStderr,
		OutputFormat: config.LogFormatText,
		Colorized:    true,
	}
	if interactive {
		l.Level = config.LoggerLevelWarning
		l.OutputFormat = logconsole.Format
	}
	return l
}
