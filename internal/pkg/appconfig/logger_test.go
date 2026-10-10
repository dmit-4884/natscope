// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package appconfig_test

import (
	"testing"

	"github.com/altessa-s/go-atlas/config/observability"

	"github.com/dmit-4884/natscope/internal/pkg/appconfig"
	"github.com/dmit-4884/natscope/internal/pkg/logconsole"
)

func TestLoggerDefaultsInteractive(t *testing.T) {
	l := appconfig.LoggerDefaults(true)

	if l.Level != observabilityconfig.LoggerLevelWarning {
		t.Errorf("Level = %q, want %q", l.Level, observabilityconfig.LoggerLevelWarning)
	}
	if l.OutputFormat != logconsole.Format {
		t.Errorf("OutputFormat = %q, want %q", l.OutputFormat, logconsole.Format)
	}
	if l.Output != observabilityconfig.LoggerConsoleOutputStderr {
		t.Errorf("Output = %q, want %q", l.Output, observabilityconfig.LoggerConsoleOutputStderr)
	}
	if !l.Colorized {
		t.Error("Colorized = false, want true")
	}
}

func TestLoggerDefaultsNonInteractive(t *testing.T) {
	l := appconfig.LoggerDefaults(false)

	if l.Level != observabilityconfig.LoggerLevelInfo {
		t.Errorf("Level = %q, want %q", l.Level, observabilityconfig.LoggerLevelInfo)
	}
	if l.OutputFormat != observabilityconfig.LogFormatText {
		t.Errorf("OutputFormat = %q, want %q", l.OutputFormat, observabilityconfig.LogFormatText)
	}
	if l.Output != observabilityconfig.LoggerConsoleOutputStderr {
		t.Errorf("Output = %q, want %q", l.Output, observabilityconfig.LoggerConsoleOutputStderr)
	}
}

func TestLoadSeedsLoggerDefaults(t *testing.T) {
	cfg, err := appconfig.Load("", appconfig.LoggerDefaults(true))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Logger.Level != observabilityconfig.LoggerLevelWarning {
		t.Errorf("Level = %q, want %q", cfg.Logger.Level, observabilityconfig.LoggerLevelWarning)
	}
	if cfg.Logger.OutputFormat != logconsole.Format {
		t.Errorf("OutputFormat = %q, want %q", cfg.Logger.OutputFormat, logconsole.Format)
	}
	if cfg.Logger.Output != observabilityconfig.LoggerConsoleOutputStderr {
		t.Errorf("Output = %q, want %q", cfg.Logger.Output, observabilityconfig.LoggerConsoleOutputStderr)
	}
}

func TestLoadEnvOverridesLoggerSeed(t *testing.T) {
	t.Setenv("LOGGER__LEVEL", "debug")
	t.Setenv("LOGGER__OUTPUT_FORMAT", "json")

	cfg, err := appconfig.Load("", appconfig.LoggerDefaults(true))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Logger.Level != observabilityconfig.LoggerLevelDebug {
		t.Errorf("Level = %q, want %q", cfg.Logger.Level, observabilityconfig.LoggerLevelDebug)
	}
	if cfg.Logger.OutputFormat != observabilityconfig.LogFormatJSON {
		t.Errorf("OutputFormat = %q, want %q", cfg.Logger.OutputFormat, observabilityconfig.LogFormatJSON)
	}
	if cfg.Logger.Output != observabilityconfig.LoggerConsoleOutputStderr {
		t.Errorf("Output = %q, want %q", cfg.Logger.Output, observabilityconfig.LoggerConsoleOutputStderr)
	}
}
