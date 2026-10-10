// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package run

import (
	"fmt"
	"slices"
	"strings"

	"github.com/altessa-s/go-atlas/config/observability"

	"github.com/dmit-4884/natscope/internal/pkg/logconsole"
)

var (
	logLevels = []string{
		observabilityconfig.LoggerLevelError,
		observabilityconfig.LoggerLevelWarning,
		observabilityconfig.LoggerLevelInfo,
		observabilityconfig.LoggerLevelDebug,
		observabilityconfig.LoggerLevelNone,
	}
	logFormats = []string{
		logconsole.Format,
		observabilityconfig.LogFormatText,
		observabilityconfig.LogFormatJSON,
	}
)

func applyLogFlags(cfg *observabilityconfig.Logger, level, format string) error {
	if level != "" {
		if !slices.Contains(logLevels, level) {
			return fmt.Errorf("invalid log level %q: expected one of %s", level, strings.Join(logLevels, ", "))
		}
		cfg.Level = level
	}
	if format != "" {
		if !slices.Contains(logFormats, format) {
			return fmt.Errorf("invalid log format %q: expected one of %s", format, strings.Join(logFormats, ", "))
		}
		cfg.OutputFormat = format
	}
	return nil
}
