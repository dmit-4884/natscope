// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package appconfig

import (
	"github.com/altessa-s/go-atlas/config/observability"

	"github.com/dmit-4884/natscope/internal/pkg/logconsole"
)

// LoggerDefaults returns the logger defaults seeded before the config file and environment apply.
func LoggerDefaults(interactive bool) *observabilityconfig.Logger {
	l := &observabilityconfig.Logger{
		Level:        observabilityconfig.LoggerLevelInfo,
		Output:       observabilityconfig.LoggerConsoleOutputStderr,
		OutputFormat: observabilityconfig.LogFormatText,
		Colorized:    true,
	}
	if interactive {
		l.Level = observabilityconfig.LoggerLevelWarning
		l.OutputFormat = logconsole.Format
	}
	return l
}
